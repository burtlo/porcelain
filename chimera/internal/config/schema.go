package config

import (
	"github.com/lynn/porcelain/chimera/chimera-indexer/adapter"
)

// chimeraDoc is the YAML shape for config/chimera.yaml.
type chimeraDoc struct {
	InternalEmbedding internalEmbeddingDoc `yaml:"internal_embedding"`
	Supervisor        supervisorDoc        `yaml:"supervisor"`

	Gateway struct {
		Enabled    *bool  `yaml:"enabled"`
		Semver     string `yaml:"semver"`
		ListenPort int    `yaml:"listen_port"`
		ListenHost string `yaml:"listen_host"`
		LogLevel   string `yaml:"log_level"`
		LogWitness struct {
			PayloadSampleMaxChars     *int  `yaml:"payload_sample_max_chars"`
			ForcePayloadSampleAtDebug *bool `yaml:"force_payload_sample_at_debug"`
		} `yaml:"log_witness"`
		Timeouts struct {
			BrokerMS int `yaml:"broker_ms"`
			ChatMS   int `yaml:"chat_ms"`
		} `yaml:"timeouts"`
		Catalog struct {
			PollMS int `yaml:"poll_ms"`
		} `yaml:"catalog"`
		Auth struct {
			APIKeys string `yaml:"api_keys"`
		} `yaml:"auth"`
		Metrics struct {
			Enabled       *bool  `yaml:"enabled"`
			SQLitePath    string `yaml:"sqlite_path"`
			MigrationsDir string `yaml:"migrations_dir"`
		} `yaml:"metrics"`
	} `yaml:"gateway"`

	Broker brokerBlock `yaml:"broker"`

	Operator struct {
		SQLitePath    string `yaml:"sqlite_path"`
		MigrationsDir string `yaml:"migrations_dir"`
	} `yaml:"operator"`

	Vectorstore vectorstoreDoc `yaml:"vectorstore"`

	Search searchDoc `yaml:"search"`

	Indexer indexerYAML `yaml:"indexer"`

	OperatorLogs struct {
		IndexerPinnedLinesMax int `yaml:"indexer_pinned_lines_max"`
	} `yaml:"operator_logs"`

	MCP mcpDoc `yaml:"mcp"`
}

type mcpDoc struct {
	CatalogMaxTools int               `yaml:"catalog_max_tools"`
	CallTimeoutMS   int               `yaml:"call_timeout_ms"`
	Servers         []mcpServerDoc    `yaml:"servers"`
	Assistants      []mcpAssistantDoc `yaml:"assistants"`
}

type mcpServerDoc struct {
	ID           string   `yaml:"id"`
	Disabled     *bool    `yaml:"disabled"`
	AutoStart    *bool    `yaml:"auto_start"`
	Transport    string   `yaml:"transport"`
	Command      string   `yaml:"command"`
	Args         []string `yaml:"args"`
	EnvAllowlist []string `yaml:"env_allowlist"`
	URL          string   `yaml:"url"`
}

type mcpAssistantDoc struct {
	AssistantID string             `yaml:"assistant_id"`
	Servers     []mcpServerBindDoc `yaml:"servers"`
}

type mcpServerBindDoc struct {
	ServerID string          `yaml:"server_id"`
	Tools    map[string]bool `yaml:"tools"`
}

type supervisorDoc struct {
	Listen              string   `yaml:"listen"`
	LogLevel            string   `yaml:"log_level"`
	LogJSON             *bool    `yaml:"log_json"`
	Services            []string `yaml:"services"`
	GatewayBin          string   `yaml:"gateway_bin"`
	GatewayListen       string   `yaml:"gateway_listen"`
	BrokerBin           string   `yaml:"broker_bin"`
	BrokerListen        string   `yaml:"broker_listen"`
	BrokerEndpoint      string   `yaml:"broker_endpoint"`
	BrokerDataDir       string   `yaml:"broker_data_dir"`
	VectorstoreBin      string   `yaml:"vectorstore_bin"`
	VectorstoreListen   string   `yaml:"vectorstore_listen"`
	VectorstoreEndpoint string   `yaml:"vectorstore_endpoint"`
	VectorstoreDataPath string   `yaml:"vectorstore_data_path"`
	ShutdownTimeoutMS   int      `yaml:"shutdown_timeout_ms"`
	TerminateWaitMS     int      `yaml:"terminate_wait_ms"`
}

type brokerBlock struct {
	Enabled   *bool  `yaml:"enabled"`
	URL       string `yaml:"url"`
	APIKeyEnv string `yaml:"api_key_env"`
	APIKey    string `yaml:"api_key"`
	LogLevel  string `yaml:"log_level"`
	HealthURL string `yaml:"health_url"`
	Models    struct {
		FreeTier string `yaml:"free_tier"`
		Limits   string `yaml:"limits"`
	} `yaml:"models"`
}

// vectorstoreDoc is the YAML shape for chimera-vectorstore.
type vectorstoreDoc struct {
	Enabled  *bool  `yaml:"enabled"`
	URL      string `yaml:"url"`
	APIKey   string `yaml:"api_key"`
	LogLevel string `yaml:"log_level"`
}

// searchDoc is the YAML shape for the search platform block.
type searchDoc struct {
	Enabled   *bool `yaml:"enabled"`
	Embedding struct {
		BaseURL string `yaml:"base_url"`
		Path    string `yaml:"path"`
		Model   string `yaml:"model"`
		Dim     int    `yaml:"dim"`
	} `yaml:"embedding"`
	Chunking struct {
		Size    int `yaml:"size"`
		Overlap int `yaml:"overlap"`
	} `yaml:"chunking"`
	RetrievalDefaults struct {
		TopK           int     `yaml:"top_k"`
		ScoreThreshold float64 `yaml:"score_threshold"`
	} `yaml:"retrieval_defaults"`
	Ingest struct {
		MaxBytes          int64 `yaml:"max_bytes"`
		MaxWholeFileBytes int64 `yaml:"max_whole_file_bytes"`
	} `yaml:"ingest"`
	Defaults struct {
		ProjectID string `yaml:"project_id"`
		FlavorID  string `yaml:"flavor_id"`
	} `yaml:"defaults"`
	Coherence struct {
		Mode string `yaml:"mode"`
	} `yaml:"coherence"`
	Tooling struct {
		Enabled         *bool `yaml:"enabled"`
		CacheTTLSeconds int   `yaml:"expansion_cache_ttl_seconds"`
		CacheMaxEntries int   `yaml:"expansion_cache_max_entries"`
	} `yaml:"tooling"`
}

// indexerYAML holds suite flags plus inline FileConfig keys and optional overlay path.
type indexerYAML struct {
	Enabled    *bool  `yaml:"enabled"`
	ConfigPath string `yaml:"config_path"`
	Bin        string `yaml:"bin"`
	LogJSON    *bool  `yaml:"log_json"`

	adapter.FileConfig `yaml:",inline"`
}

func (d searchDoc) toRAGDoc() ragDoc {
	r := ragDoc{Enabled: d.Enabled}
	r.Embedding = d.Embedding
	r.Chunking = d.Chunking
	r.Ingest = d.Ingest
	r.Defaults = d.Defaults
	r.Coherence = d.Coherence
	r.Tooling = d.Tooling
	r.Retrieval.TopK = d.RetrievalDefaults.TopK
	r.Retrieval.ScoreThreshold = d.RetrievalDefaults.ScoreThreshold
	return r
}

func boolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
