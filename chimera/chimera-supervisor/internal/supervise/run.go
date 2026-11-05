package supervise

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	svconfig "github.com/lynn/porcelain/chimera/chimera-supervisor/internal/config"
	"github.com/lynn/porcelain/chimera/chimera-supervisor/internal/control"
	"github.com/lynn/porcelain/chimera/chimera-supervisor/internal/supervisorline"
	"github.com/lynn/porcelain/chimera/internal/brokerclient"
	gwconfig "github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/chimera/internal/logfmt"
	"github.com/lynn/porcelain/chimera/internal/servicelogs"
	"github.com/lynn/porcelain/chimera/internal/tokens"
)

// Run supervises gateway, broker, vectorstore, optional embed, and optional indexer until ctx is canceled.
func Run(ctx context.Context, cfg svconfig.Config, version, commit string) error {
	path := strings.TrimSpace(cfg.ConfigPath)
	if path == "" {
		var err error
		path, err = gwconfig.ResolveChimeraConfigPath()
		if err != nil {
			return svconfig.Exitf(2, "%v", err)
		}
	}

	res, err := gwconfig.LoadChimeraYAML(path, nil)
	if err != nil {
		return svconfig.Exitf(1, "load chimera.yaml: %v", err)
	}
	applySupervisorYAMLOverrides(&cfg, res)

	logStore := servicelogs.New(servicelogs.DefaultMaxLines)
	logLevel := resolveCollectorLogLevel(res)
	supSink := LogSink(logStore.Writer(servicelogs.SourceChimeraSupervisor), supervisorline.NewWriter, logLevel)
	logJSON := cfg.LogJSON
	if res.SupervisorLogJSON {
		logJSON = true
	}
	log := buildLogger(supSink, logLevel, logJSON)
	if logJSON {
		_ = os.Setenv(logfmt.EnvLogJSON, "1")
	}
	embedEndpoint := strings.TrimSpace(cfg.EmbedEndpoint)
	if res.InternalEmbedding.Enabled && embedEndpoint == "" {
		embedEndpoint, err = res.InternalEmbedding.Endpoint()
		if err != nil {
			return svconfig.Exitf(1, "%v", err)
		}
	}

	log.Info("supervisor startup seed", "msg", "chimera-supervisor.startup.seed")
	rootCtx, stopRoot := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopRoot()

	bootstrap := false
	if strings.TrimSpace(res.TokensPath) != "" {
		bootstrap = tokens.IsBootstrapMode(res.TokensPath)
	}

	launch, lerr := res.ResolveSupervisorServices()
	if lerr != nil {
		return svconfig.Exitf(1, "%v", lerr)
	}
	want := map[string]bool{}
	for _, s := range launch {
		want[s] = true
	}

	vectorstoreWrapperBin := strings.TrimSpace(cfg.VectorstoreBin)
	if !want[gwconfig.ServiceVectorstore] {
		vectorstoreWrapperBin = ""
	}
	controlState := control.NewState()
	controlState.SetVersions(version, commit)
	controlState.SetRequired(want[gwconfig.ServiceBroker] || want[gwconfig.ServiceGateway], want[gwconfig.ServiceVectorstore])
	controlState.SetEmbedRequired(res.InternalEmbedding.Enabled)
	controlState.SetEmbedEndpoint(embedEndpoint)
	controlState.SetEndpoints(strings.TrimSpace(cfg.BrokerEndpoint), strings.TrimSpace(cfg.VectorstoreEndpoint))
	controlState.SetOperatorUI(gatewayPublicURLFromResolved(res), bootstrap)
	controlListen := strings.TrimSpace(cfg.Listen)
	if controlListen == "" {
		controlListen = "127.0.0.1:7710"
	}
	controlBaseURL := fmt.Sprintf("http://%s", controlListen)
	controlSrv := &http.Server{Addr: controlListen, Handler: control.Handler(controlState, logStore, stopRoot)}
	controlLn, controlErr := net.Listen("tcp", controlListen)
	if controlErr != nil {
		return svconfig.Exitf(1, "listen %s: %v", controlListen, controlErr)
	}
	go func() {
		if err := controlSrv.Serve(controlLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("supervisor control server exit", "msg", "chimera-supervisor.control.server_error", "listen", controlListen, "err", err)
		}
	}()
	defer func() {
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = controlSrv.Shutdown(shCtx)
	}()

	vectorstoreReadyzURL := ""
	if vectorstoreWrapperBin != "" {
		vectorstoreReadyzURL = fmt.Sprintf("http://%s/readyz", strings.TrimSpace(cfg.VectorstoreListen))
	}
	embedReadyzURL := ""
	if res.InternalEmbedding.Enabled {
		embedReadyzURL = fmt.Sprintf("http://%s/readyz", strings.TrimSpace(cfg.EmbedListen))
	}
	gatewayReadyzURL := fmt.Sprintf("http://%s/readyz", strings.TrimSpace(cfg.GatewayListen))
	brokerReadyzURL := fmt.Sprintf("http://%s/readyz", strings.TrimSpace(cfg.BrokerListen))

	var (
		gatewayProc     *exec.Cmd
		gatewayWaitErr  chan error
		vectorstoreProc *exec.Cmd
		vectorstoreWait chan error
		embedProc       *exec.Cmd
		embedWait       chan error
		brokerProc      *exec.Cmd
		brokerWaitErr   chan error
		indexerProc     *exec.Cmd
		indexerWait     chan error
	)

	indexerCtx, stopIndexer := context.WithCancel(ctx)
	var supervisedShutdownOnce sync.Once
	stopChildrenGraceful := func() {
		supervisedShutdownOnce.Do(func() {
			stopIndexer()
			shutdownGrace := cfg.ShutdownTimeout
			if cfg.TerminateWait > shutdownGrace {
				shutdownGrace = cfg.TerminateWait
			}
			ShutdownChildren(log, shutdownGrace,
				Child{Name: "gateway", Cmd: gatewayProc, WaitCh: gatewayWaitErr},
				Child{Name: "vectorstore", Cmd: vectorstoreProc, WaitCh: vectorstoreWait},
				Child{Name: "embed", Cmd: embedProc, WaitCh: embedWait},
				Child{Name: "broker", Cmd: brokerProc, WaitCh: brokerWaitErr},
				Child{Name: "indexer", Cmd: indexerProc, WaitCh: indexerWait},
			)
			log.Info("supervised shutdown complete", "msg", "chimera-supervisor.shutdown.children_done")
		})
	}
	stopChildrenFast := func() {
		supervisedShutdownOnce.Do(func() {
			stopIndexer()
			KillWrapperFamilies(gatewayProc, brokerProc, vectorstoreProc, embedProc)
		})
	}

	if !bootstrap {
		if want[gwconfig.ServiceVectorstore] && vectorstoreWrapperBin != "" {
			if err := startVectorstoreChild(cfg, res, controlBaseURL, logStore, logLevel, log, controlState, vectorstoreWrapperBin, &vectorstoreProc, &vectorstoreWait, vectorstoreReadyzURL, stopChildrenFast); err != nil {
				return err
			}
		}
		if res.InternalEmbedding.Enabled {
			if strings.TrimSpace(cfg.EmbedBin) == "" {
				return svconfig.Exitf(1, "internal_embedding.enabled but no chimera-embed wrapper was configured")
			}
			if err := startEmbedChild(cfg, res, embedEndpoint, controlBaseURL, logStore, logLevel, log, controlState, &embedProc, &embedWait, embedReadyzURL, stopChildrenFast); err != nil {
				return err
			}
		}
		if want[gwconfig.ServiceBroker] {
			if err := startBrokerChild(cfg, res, controlBaseURL, logStore, logLevel, log, controlState, &brokerProc, &brokerWaitErr, brokerReadyzURL, vectorstoreWait, stopChildrenFast); err != nil {
				return err
			}
		}
		if want[gwconfig.ServiceGateway] {
			if err := startGatewayChild(cfg, path, controlBaseURL, logStore, logLevel, log, controlState, &gatewayProc, &gatewayWaitErr, gatewayReadyzURL, stopChildrenFast); err != nil {
				return err
			}
		}
		if want[gwconfig.ServiceIndexer] {
			startIndexerChild(res, cfg, path, controlBaseURL, logStore, logLevel, log, indexerCtx, &indexerProc, &indexerWait)
		} else if log != nil {
			log.Info("indexer not in supervisor.services", "msg", "chimera-supervisor.indexer.skipped",
				"indexer_enabled", res.IndexerEnabled)
		}
	} else {
		// Bootstrap: gateway-only loopback setup surface until api-keys.yaml exists.
		if err := startGatewayChild(cfg, path, controlBaseURL, logStore, logLevel, log, controlState, &gatewayProc, &gatewayWaitErr, gatewayReadyzURL, stopChildrenFast); err != nil {
			return err
		}
	}

	go func() {
		<-rootCtx.Done()
		log.Info("received shutdown signal", "msg", "chimera-supervisor.shutdown.signal_received")
		log.Info("shutting down gracefully", "msg", "chimera-supervisor.shutdown.graceful_start")
		stopChildrenGraceful()
	}()
	if want[gwconfig.ServiceGateway] {
		brokerclient.RunSupervisedChildHealthMonitor(rootCtx, log, "gateway", gatewayReadyzURL, 15*time.Second, 30*time.Second, !cfg.NoWaitGateway)
	}
	if want[gwconfig.ServiceBroker] {
		brokerclient.RunSupervisedChildHealthMonitor(rootCtx, log, "broker", brokerReadyzURL, 15*time.Second, 30*time.Second, !cfg.NoWaitBroker)
	}
	if vectorstoreReadyzURL != "" {
		brokerclient.RunSupervisedChildHealthMonitor(rootCtx, log, "vectorstore", vectorstoreReadyzURL, 15*time.Second, 30*time.Second, !cfg.NoWaitVectorstore)
	}
	if embedReadyzURL != "" {
		brokerclient.RunSupervisedChildHealthMonitor(rootCtx, log, "embed", embedReadyzURL, 15*time.Second, 30*time.Second, !cfg.NoWaitEmbed)
	}
	<-rootCtx.Done()
	stopChildrenGraceful()
	return nil
}

// applySupervisorYAMLOverrides fills empty CLI config fields from chimera.yaml supervisor:.
func applySupervisorYAMLOverrides(cfg *svconfig.Config, res *gwconfig.Resolved) {
	if cfg == nil || res == nil {
		return
	}
	if strings.TrimSpace(cfg.Listen) == "" && res.SupervisorListen != "" {
		cfg.Listen = res.SupervisorListen
	}
	if strings.TrimSpace(cfg.GatewayBin) == "" && res.SupervisorGatewayBin != "" {
		cfg.GatewayBin = res.SupervisorGatewayBin
	}
	if strings.TrimSpace(cfg.GatewayListen) == "" && res.SupervisorGatewayListen != "" {
		cfg.GatewayListen = res.SupervisorGatewayListen
	}
	if strings.TrimSpace(cfg.BrokerBin) == "" && res.SupervisorBrokerBin != "" {
		cfg.BrokerBin = res.SupervisorBrokerBin
	}
	if strings.TrimSpace(cfg.BrokerListen) == "" && res.SupervisorBrokerListen != "" {
		cfg.BrokerListen = res.SupervisorBrokerListen
	}
	if strings.TrimSpace(cfg.BrokerEndpoint) == "" && res.SupervisorBrokerEndpoint != "" {
		cfg.BrokerEndpoint = res.SupervisorBrokerEndpoint
	}
	if strings.TrimSpace(cfg.BrokerDataDir) == "" && res.SupervisorBrokerDataDir != "" {
		cfg.BrokerDataDir = res.SupervisorBrokerDataDir
	}
	if strings.TrimSpace(cfg.VectorstoreBin) == "" && res.SupervisorVectorstoreBin != "" {
		cfg.VectorstoreBin = res.SupervisorVectorstoreBin
	}
	if strings.TrimSpace(cfg.VectorstoreListen) == "" && res.SupervisorVectorstoreListen != "" {
		cfg.VectorstoreListen = res.SupervisorVectorstoreListen
	}
	if strings.TrimSpace(cfg.VectorstoreEndpoint) == "" && res.SupervisorVectorstoreEndpoint != "" {
		cfg.VectorstoreEndpoint = res.SupervisorVectorstoreEndpoint
	}
	if strings.TrimSpace(cfg.VectorstoreDataPath) == "" && res.SupervisorVectorstoreDataPath != "" {
		cfg.VectorstoreDataPath = res.SupervisorVectorstoreDataPath
	}
	if res.SupervisorShutdownTimeoutMS > 0 {
		cfg.ShutdownTimeout = time.Duration(res.SupervisorShutdownTimeoutMS) * time.Millisecond
	}
	if res.SupervisorTerminateWaitMS > 0 {
		cfg.TerminateWait = time.Duration(res.SupervisorTerminateWaitMS) * time.Millisecond
	}
}
