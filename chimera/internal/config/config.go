package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-indexer/adapter"
	"github.com/lynn/porcelain/chimera/internal/providerfreetier"
	"github.com/lynn/porcelain/chimera/internal/providerlimits"
	"github.com/lynn/porcelain/internal/naming"
	"gopkg.in/yaml.v3"
)

// Service names used in supervisor.services and suite membership.
const (
	ServiceGateway     = "gateway"
	ServiceBroker      = "broker"
	ServiceVectorstore = "vectorstore"
	ServiceIndexer     = "indexer"
)

// Resolved matches TypeScript ResolvedGatewayConfig (src/config.ts).
type Resolved struct {
	Semver            string
	ListenPort        int
	ListenHost        string
	LogLevel          string
	BrokerLogLevel    string // supervised chimera-broker wrapper (broker.log_level).
	UpstreamBaseURL   string
	UpstreamAPIKeyEnv string
	// UpstreamAPIKey is the Bearer token from chimera.yaml (broker.api_key). Non-empty process env named by UpstreamAPIKeyEnv overrides at runtime.
	UpstreamAPIKey    string
	HealthUpstreamURL string
	HealthTimeoutMs   int
	ChatTimeoutMs     int
	// AvailableModelsPollMs is the period for the BiFrost `/v1/models` catalog poller.
	// 0 disables polling (one-shot startup refresh only).
	AvailableModelsPollMs int
	TokensPath            string
	ChimeraYAMLPath       string // path to chimera.yaml
	// ProviderFreeTierPath is the resolved filesystem path to provider-free-tier.yaml.
	ProviderFreeTierPath  string
	ProviderFreeTierSpec  *providerfreetier.Spec
	MetricsEnabled        bool
	MetricsSQLitePath     string
	MetricsMigrationsDir  string
	OperatorSQLitePath    string
	OperatorMigrationsDir string
	ProviderLimitsPath    string
	ProviderLimitsSpec    *providerlimits.Config
	// RAG holds search-platform settings (YAML search:); Enabled gates ingest, indexer REST, retrieval, and the /health vectorstore probe.
	RAG RAG
	// InternalEmbedding configures the opt-in supervised chimera-embed runtime.
	InternalEmbedding InternalEmbedding

	// Suite membership (enabled does not start processes).
	GatewayEnabled     bool
	BrokerEnabled      bool
	VectorstoreEnabled bool
	IndexerEnabled     bool

	// Supervisor* from supervisor: block (CLI flags may override at runtime).
	SupervisorLogLevel            string // collector gate for LogSink
	SupervisorLogJSON             bool
	SupervisorListen              string
	SupervisorServices            []string // nil/empty → default all enabled services
	SupervisorGatewayBin          string
	SupervisorGatewayListen       string
	SupervisorBrokerBin           string
	SupervisorBrokerListen        string
	SupervisorBrokerEndpoint      string
	SupervisorBrokerDataDir       string
	SupervisorVectorstoreBin      string
	SupervisorVectorstoreListen   string
	SupervisorVectorstoreEndpoint string
	SupervisorVectorstoreDataPath string
	SupervisorShutdownTimeoutMS   int
	SupervisorTerminateWaitMS     int

	// Indexer inline FileConfig + overlay / materialize paths.
	IndexerFileConfig       adapter.FileConfig
	IndexerOverlayPath      string // absolute optional operator overlay (config_path)
	IndexerMaterializedPath string // absolute path written for --config
	IndexerBin              string
	IndexerLogJSON          bool

	// IndexerSupervised* retained for UI/API compatibility (derived from suite + search).
	IndexerSupervisedEnabled              bool
	IndexerSupervisedBin                  string
	IndexerSupervisedConfigPath           string // materialize path (child --config)
	IndexerSupervisedStartWhenRAGDisabled bool   // always false; retained for DTO compat
	IndexerSupervisedLogJSON              bool

	WitnessSampleMaxChars             int
	WitnessSampleForceAtDebug         bool
	OperatorLogsIndexerPinnedLinesMax int

	MCP MCP
}

// ShouldEmitPayloadSample reports whether conversation.payload.sample may be emitted.
func (r *Resolved) ShouldEmitPayloadSample() bool {
	if r == nil {
		return false
	}
	ll := strings.ToLower(strings.TrimSpace(r.LogLevel))
	if ll == "trace" {
		return true
	}
	return r.WitnessSampleForceAtDebug && (ll == "debug" || ll == "trace")
}

// WitnessSampleMaxRunes returns the configured max runes per head/tail for payload samples.
func (r *Resolved) WitnessSampleMaxRunes() int {
	if r == nil || r.WitnessSampleMaxChars <= 0 {
		return 256
	}
	if r.WitnessSampleMaxChars > 4096 {
		return 4096
	}
	if r.WitnessSampleMaxChars < 32 {
		return 32
	}
	return r.WitnessSampleMaxChars
}

// DefaultSupervisorServices returns the launch list when supervisor.services is omitted.
func (r *Resolved) DefaultSupervisorServices() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, 4)
	if r.VectorstoreEnabled {
		out = append(out, ServiceVectorstore)
	}
	if r.BrokerEnabled {
		out = append(out, ServiceBroker)
	}
	if r.GatewayEnabled {
		out = append(out, ServiceGateway)
	}
	if r.IndexerEnabled {
		out = append(out, ServiceIndexer)
	}
	return out
}

// ResolveSupervisorServices returns the effective launch list, or an error if a listed service is suite-disabled.
func (r *Resolved) ResolveSupervisorServices() ([]string, error) {
	if r == nil {
		return nil, fmt.Errorf("nil resolved config")
	}
	list := r.SupervisorServices
	if len(list) == 0 {
		return r.DefaultSupervisorServices(), nil
	}
	enabled := map[string]bool{
		ServiceGateway:     r.GatewayEnabled,
		ServiceBroker:      r.BrokerEnabled,
		ServiceVectorstore: r.VectorstoreEnabled,
		ServiceIndexer:     r.IndexerEnabled,
	}
	out := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, raw := range list {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || seen[name] {
			continue
		}
		ok, known := enabled[name]
		if !known {
			return nil, fmt.Errorf("supervisor.services: unknown service %q", raw)
		}
		if !ok {
			return nil, fmt.Errorf("supervisor.services: %q is listed but %s.enabled is false", name, name)
		}
		seen[name] = true
		out = append(out, name)
	}
	return out, nil
}

// ServiceInLaunchList reports whether name is in the effective supervisor launch set.
func (r *Resolved) ServiceInLaunchList(name string) bool {
	list, err := r.ResolveSupervisorServices()
	if err != nil {
		return false
	}
	name = strings.ToLower(strings.TrimSpace(name))
	for _, s := range list {
		if s == name {
			return true
		}
	}
	return false
}

const (
	defaultSemver                = "0.1.0"
	defaultListenPort            = 3000
	defaultListenHost            = "0.0.0.0"
	defaultLogLevel              = "info"
	defaultBaseURL               = "http://chimera-broker:8080"
	defaultAPIKeyEnv             = naming.EnvBrokerAPIKeyTarget
	defaultHealthTimeoutMs       = 5000
	defaultChatTimeoutMs         = 300_000
	defaultAvailableModelsPollMs = 30_000
	defaultIndexerPinnedLinesMax = 64
)

// LoadChimeraYAML reads and parses chimera.yaml at filePath.
func LoadChimeraYAML(filePath string, log *slog.Logger) (*Resolved, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var doc chimeraDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse chimera yaml: %w", err)
	}

	semver := doc.Gateway.Semver
	if semver == "" {
		semver = defaultSemver
	}

	upBase := strings.TrimSuffix(strings.TrimSpace(doc.Broker.URL), "/")
	if upBase == "" {
		upBase = strings.TrimSuffix(defaultBaseURL, "/")
	}

	apiKeyEnv := doc.Broker.APIKeyEnv
	if apiKeyEnv == "" {
		apiKeyEnv = defaultAPIKeyEnv
	}
	apiKey := strings.TrimSpace(doc.Broker.APIKey)

	healthURL := strings.TrimSpace(doc.Broker.HealthURL)
	if healthURL == "" {
		healthURL = upBase + "/health"
	}

	baseDir := filepath.Dir(filePath)

	apiKeysRel := strings.TrimSpace(doc.Gateway.Auth.APIKeys)
	if apiKeysRel == "" {
		apiKeysRel = "./" + naming.APIKeysFileTarget
	}
	tokensPath := resolveBeside(baseDir, apiKeysRel)

	ftRel := strings.TrimSpace(doc.Broker.Models.FreeTier)
	if ftRel == "" {
		ftRel = "./provider-free-tier.yaml"
	}
	ftPath := resolveBeside(baseDir, ftRel)
	var ftSpec *providerfreetier.Spec
	if st, err := os.Stat(ftPath); err == nil && !st.IsDir() {
		s, err := providerfreetier.Load(ftPath)
		if err != nil {
			if log != nil {
				log.Error("provider free tier yaml invalid", "msg", "chat.provider_limits.config_invalid", "path", ftPath, "err", err)
			}
		} else {
			ftSpec = s
		}
	} else if err != nil && !os.IsNotExist(err) && log != nil {
		log.Warn("provider free tier path not stat-able", "msg", "chat.provider_limits.config_missing", "path", ftPath, "err", err)
	}

	limitsRel := strings.TrimSpace(doc.Broker.Models.Limits)
	if limitsRel == "" {
		limitsRel = "./provider-model-limits.yaml"
	}
	limitsPath := resolveBeside(baseDir, limitsRel)
	limitsSpec, err := providerlimits.LoadOrEmpty(limitsPath)
	if err != nil {
		if log != nil {
			log.Error("provider-model-limits.yaml invalid; using empty spec (no enforcement)", "msg", "chat.provider_limits.config_invalid", "path", limitsPath, "err", err)
		}
		limitsSpec = &providerlimits.Config{}
	}

	listenPort := doc.Gateway.ListenPort
	if listenPort == 0 {
		listenPort = defaultListenPort
	}
	listenHost := doc.Gateway.ListenHost
	if listenHost == "" {
		listenHost = defaultListenHost
	}

	ht := doc.Gateway.Timeouts.BrokerMS
	if ht == 0 {
		ht = defaultHealthTimeoutMs
	}
	ct := doc.Gateway.Timeouts.ChatMS
	if ct == 0 {
		ct = defaultChatTimeoutMs
	}
	availPoll := doc.Gateway.Catalog.PollMS
	if availPoll == 0 {
		availPoll = defaultAvailableModelsPollMs
	}
	if availPoll < 0 {
		availPoll = 0
	}

	metricsEnabled := true
	if doc.Gateway.Metrics.Enabled != nil {
		metricsEnabled = *doc.Gateway.Metrics.Enabled
	}
	sqliteRel := strings.TrimSpace(doc.Gateway.Metrics.SQLitePath)
	if sqliteRel == "" {
		sqliteRel = filepath.Join("..", "data", "gateway", "metrics.sqlite")
	}
	metricsSQLite := resolveBeside(baseDir, sqliteRel)
	migRel := strings.TrimSpace(doc.Gateway.Metrics.MigrationsDir)
	if migRel == "" {
		migRel = filepath.Join("..", "migrations", "chimera-gateway", "metrics")
	}
	metricsMig := resolveBeside(baseDir, migRel)

	opSqliteRel := strings.TrimSpace(doc.Operator.SQLitePath)
	if opSqliteRel == "" {
		opSqliteRel = filepath.Join("..", "data", "gateway", "operator.sqlite")
	}
	operatorSQLite := resolveBeside(baseDir, opSqliteRel)
	opMigRel := strings.TrimSpace(doc.Operator.MigrationsDir)
	if opMigRel == "" {
		opMigRel = filepath.Join("..", "migrations", "chimera-gateway", "operator")
	}
	operatorMig := resolveBeside(baseDir, opMigRel)

	logLevel := doc.Gateway.LogLevel
	if logLevel == "" {
		logLevel = defaultLogLevel
	}
	brokerLogLevel := strings.TrimSpace(doc.Broker.LogLevel)

	witnessMax := 256
	if doc.Gateway.LogWitness.PayloadSampleMaxChars != nil && *doc.Gateway.LogWitness.PayloadSampleMaxChars > 0 {
		witnessMax = *doc.Gateway.LogWitness.PayloadSampleMaxChars
	}
	witnessForceDebug := false
	if doc.Gateway.LogWitness.ForcePayloadSampleAtDebug != nil {
		witnessForceDebug = *doc.Gateway.LogWitness.ForcePayloadSampleAtDebug
	}

	rag := doc.Search.toRAGDoc().effective(doc.Vectorstore)
	internalEmbedding := doc.InternalEmbedding.effective(baseDir)
	if err := internalEmbedding.Validate(); err != nil {
		return nil, err
	}
	applyInternalEmbeddingToRAG(&rag, internalEmbedding)
	if err := rag.Validate(); err != nil {
		if log != nil {
			log.Error("search config invalid; disabling search", "msg", "rag.config.invalid", "err", err)
		}
		rag = RAG{Enabled: false}
	}

	gatewayEnabled := boolOrDefault(doc.Gateway.Enabled, true)
	brokerEnabled := boolOrDefault(doc.Broker.Enabled, true)
	vectorstoreEnabled := boolOrDefault(doc.Vectorstore.Enabled, true)
	indexerEnabled := boolOrDefault(doc.Indexer.Enabled, true)

	idxBin := strings.TrimSpace(doc.Indexer.Bin)
	idxLogJSON := true
	if doc.Indexer.LogJSON != nil {
		idxLogJSON = *doc.Indexer.LogJSON
	}

	idxOverlayRel := strings.TrimSpace(doc.Indexer.ConfigPath)
	idxOverlayPath := ""
	if idxOverlayRel != "" {
		idxOverlayPath = resolveBeside(baseDir, idxOverlayRel)
	}

	materializedRel := filepath.Join("..", "data", "gateway", "indexer.materialized.yaml")
	materializedPath := resolveBeside(baseDir, materializedRel)

	idxFile := doc.Indexer.FileConfig

	idxPinnedMax := doc.OperatorLogs.IndexerPinnedLinesMax
	if idxPinnedMax <= 0 {
		idxPinnedMax = defaultIndexerPinnedLinesMax
	}

	supLogLevel := strings.TrimSpace(doc.Supervisor.LogLevel)
	if supLogLevel == "" {
		supLogLevel = defaultLogLevel
	}
	supLogJSON := true
	if doc.Supervisor.LogJSON != nil {
		supLogJSON = *doc.Supervisor.LogJSON
	}

	services := make([]string, 0, len(doc.Supervisor.Services))
	for _, s := range doc.Supervisor.Services {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" {
			services = append(services, s)
		}
	}

	if log != nil {
		log.Info("chimera config resolved", "msg", "gateway.startup.config_resolved",
			"filePath", filePath, "api_keys_path", tokensPath)
	}

	res := &Resolved{
		Semver:                                semver,
		ListenPort:                            listenPort,
		ListenHost:                            listenHost,
		LogLevel:                              logLevel,
		BrokerLogLevel:                        brokerLogLevel,
		UpstreamBaseURL:                       upBase,
		UpstreamAPIKeyEnv:                     apiKeyEnv,
		UpstreamAPIKey:                        apiKey,
		HealthUpstreamURL:                     healthURL,
		HealthTimeoutMs:                       ht,
		ChatTimeoutMs:                         ct,
		AvailableModelsPollMs:                 availPoll,
		TokensPath:                            tokensPath,
		ChimeraYAMLPath:                       filePath,
		ProviderFreeTierPath:                  ftPath,
		ProviderFreeTierSpec:                  ftSpec,
		MetricsEnabled:                        metricsEnabled,
		MetricsSQLitePath:                     metricsSQLite,
		MetricsMigrationsDir:                  metricsMig,
		OperatorSQLitePath:                    operatorSQLite,
		OperatorMigrationsDir:                 operatorMig,
		ProviderLimitsPath:                    limitsPath,
		ProviderLimitsSpec:                    limitsSpec,
		RAG:                                   rag,
		InternalEmbedding:                     internalEmbedding,
		GatewayEnabled:                        gatewayEnabled,
		BrokerEnabled:                         brokerEnabled,
		VectorstoreEnabled:                    vectorstoreEnabled,
		IndexerEnabled:                        indexerEnabled,
		SupervisorLogLevel:                    supLogLevel,
		SupervisorLogJSON:                     supLogJSON,
		SupervisorListen:                      strings.TrimSpace(doc.Supervisor.Listen),
		SupervisorServices:                    services,
		SupervisorGatewayBin:                  strings.TrimSpace(doc.Supervisor.GatewayBin),
		SupervisorGatewayListen:               strings.TrimSpace(doc.Supervisor.GatewayListen),
		SupervisorBrokerBin:                   strings.TrimSpace(doc.Supervisor.BrokerBin),
		SupervisorBrokerListen:                strings.TrimSpace(doc.Supervisor.BrokerListen),
		SupervisorBrokerEndpoint:              strings.TrimSpace(doc.Supervisor.BrokerEndpoint),
		SupervisorBrokerDataDir:               strings.TrimSpace(doc.Supervisor.BrokerDataDir),
		SupervisorVectorstoreBin:              strings.TrimSpace(doc.Supervisor.VectorstoreBin),
		SupervisorVectorstoreListen:           strings.TrimSpace(doc.Supervisor.VectorstoreListen),
		SupervisorVectorstoreEndpoint:         strings.TrimSpace(doc.Supervisor.VectorstoreEndpoint),
		SupervisorVectorstoreDataPath:         strings.TrimSpace(doc.Supervisor.VectorstoreDataPath),
		SupervisorShutdownTimeoutMS:           doc.Supervisor.ShutdownTimeoutMS,
		SupervisorTerminateWaitMS:             doc.Supervisor.TerminateWaitMS,
		IndexerFileConfig:                     idxFile,
		IndexerOverlayPath:                    idxOverlayPath,
		IndexerMaterializedPath:               materializedPath,
		IndexerBin:                            idxBin,
		IndexerLogJSON:                        idxLogJSON,
		IndexerSupervisedEnabled:              indexerEnabled,
		IndexerSupervisedBin:                  idxBin,
		IndexerSupervisedConfigPath:           materializedPath,
		IndexerSupervisedStartWhenRAGDisabled: false,
		IndexerSupervisedLogJSON:              idxLogJSON,
		WitnessSampleMaxChars:                 witnessMax,
		WitnessSampleForceAtDebug:             witnessForceDebug,
		OperatorLogsIndexerPinnedLinesMax:     idxPinnedMax,
		MCP:                                   parseMCPDoc(doc, log),
	}
	if _, err := res.ResolveSupervisorServices(); err != nil {
		return nil, err
	}
	return res, nil
}

func resolveBeside(baseDir, relOrAbs string) string {
	relOrAbs = strings.TrimSpace(relOrAbs)
	if filepath.IsAbs(relOrAbs) {
		return relOrAbs
	}
	return filepath.Join(baseDir, relOrAbs)
}

// ResolveChimeraConfigPath returns CHIMERA_CONFIG when set, otherwise ./config/chimera.yaml.
func ResolveChimeraConfigPath() (string, error) {
	if e := strings.TrimSpace(os.Getenv(naming.EnvChimeraConfigTarget)); e != "" {
		return filepath.Clean(e), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, naming.ChimeraConfigDirTarget, naming.ChimeraConfigFileTarget), nil
}

// ListenAddr returns "host:port" for net.Listen.
func (r *Resolved) ListenAddr() string {
	return fmt.Sprintf("%s:%d", r.ListenHost, r.ListenPort)
}
