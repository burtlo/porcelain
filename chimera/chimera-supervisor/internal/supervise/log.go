package supervise

import (
	"io"
	"log/slog"
	"os"
	"strings"

	gwconfig "github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/chimera/internal/logfmt"
	wline "github.com/lynn/porcelain/chimera/internal/wrapper/line"
)

// LogSink normalizes child stdout/stderr to JSON lines, applies minLevel, and records
// them in the ring buffer and process stdout.
func LogSink(storeWriter io.Writer, normalize func(io.Writer) io.Writer, minLevel slog.Level) io.Writer {
	sink := io.MultiWriter(
		wline.NewLevelFilterWriter(storeWriter, minLevel),
		wline.NewLevelFilterWriter(os.Stdout, minLevel),
	)
	return normalize(sink)
}

// resolveCollectorLogLevel returns the supervisor collector gate level.
// LOG_LEVEL env overrides supervisor.log_level from chimera.yaml.
func resolveCollectorLogLevel(res *gwconfig.Resolved) slog.Level {
	if e := strings.TrimSpace(os.Getenv("LOG_LEVEL")); e != "" {
		return wline.ParseLogLevel(e)
	}
	if res != nil && strings.TrimSpace(res.SupervisorLogLevel) != "" {
		return wline.ParseLogLevel(res.SupervisorLogLevel)
	}
	return slog.LevelInfo
}

// resolveLogLevel loads chimera.yaml and returns the collector gate (legacy name kept for callers).
func resolveLogLevel(gatewayPath string) slog.Level {
	res, err := gwconfig.LoadChimeraYAML(gatewayPath, nil)
	if err != nil {
		if e := strings.TrimSpace(os.Getenv("LOG_LEVEL")); e != "" {
			return wline.ParseLogLevel(e)
		}
		return slog.LevelInfo
	}
	return resolveCollectorLogLevel(res)
}

func buildLogger(w io.Writer, level slog.Level, json bool) *slog.Logger {
	return logfmt.NewLogger(w, json, level)
}
