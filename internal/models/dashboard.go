package models

import "time"

type DashboardStatus struct {
	Status          string          `json:"status"`
	ServerName      string          `json:"server_name"`
	ServerVersion   string          `json:"server_version"`
	ServiceURL      string          `json:"service_url"`
	Transport       string          `json:"transport"`
	HTTPAddr        string          `json:"http_addr,omitempty"`
	AuthMode        string          `json:"auth_mode"`
	ReadOnlyMode    string          `json:"read_only_mode"`
	UniversalTool   bool            `json:"universal_tool"`
	ProtocolVersion string          `json:"protocol_version"`
	IsSAPService    bool            `json:"is_sap_service"`
	ToolCount       int             `json:"tool_count"`
	MetadataSummary MetadataSummary `json:"metadata_summary"`
	LastLoadedAt    time.Time       `json:"last_loaded_at"`
}

type ConnectionTestResult struct {
	OK              bool            `json:"ok"`
	Message         string          `json:"message"`
	ServiceURL      string          `json:"service_url"`
	DurationMs      int64           `json:"duration_ms"`
	IsSAPService    bool            `json:"is_sap_service"`
	MetadataSummary MetadataSummary `json:"metadata_summary"`
	Version         string          `json:"version"`
}

type MetadataOverview struct {
	ServiceRoot     string                `json:"service_root"`
	SchemaNamespace string                `json:"schema_namespace"`
	ContainerName   string                `json:"container_name"`
	Version         string                `json:"version"`
	ParsedAt        time.Time             `json:"parsed_at"`
	Summary         MetadataSummary       `json:"summary"`
	EntitySets      []EntitySetSummary    `json:"entity_sets"`
	FunctionImports []FunctionImportBrief `json:"function_imports"`
}

type EntitySetSummary struct {
	Name       string `json:"name"`
	EntityType string `json:"entity_type"`
	Creatable  bool   `json:"creatable"`
	Updatable  bool   `json:"updatable"`
	Deletable  bool   `json:"deletable"`
	Searchable bool   `json:"searchable"`
	Pageable   bool   `json:"pageable"`
}

type FunctionImportBrief struct {
	Name           string `json:"name"`
	HTTPMethod     string `json:"http_method"`
	ReturnType     string `json:"return_type,omitempty"`
	ParameterCount int    `json:"parameter_count"`
	IsAction       bool   `json:"is_action"`
}

type DashboardSettingItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type DashboardEndpoint struct {
	Name        string `json:"name"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type DashboardSettings struct {
	DashboardPath            string                 `json:"dashboard_path"`
	DocumentationPath        string                 `json:"documentation_path"`
	DiagnosticsPath          string                 `json:"diagnostics_path"`
	Transport                string                 `json:"transport"`
	HTTPAddr                 string                 `json:"http_addr,omitempty"`
	ConfigItems              []DashboardSettingItem `json:"config_items"`
	Endpoints                []DashboardEndpoint    `json:"endpoints"`
	Notes                    []string               `json:"notes"`
	SupportsLiveConnectivity bool                   `json:"supports_live_connectivity"`
}

type DashboardDiagnostics struct {
	GeneratedAt time.Time          `json:"generated_at"`
	Status      *DashboardStatus   `json:"status"`
	Settings    *DashboardSettings `json:"settings"`
	Metadata    *MetadataOverview  `json:"metadata"`
}

type DashboardConnection struct {
	Name           string `json:"name"`
	SystemName     string `json:"system_name"`
	ServiceURL     string `json:"service_url"`
	SafeServiceURL string `json:"safe_service_url"`
	Client         string `json:"client,omitempty"`
	Username       string `json:"username"`
	AccessMode     string `json:"access_mode"`
	Connected      bool   `json:"connected"`
}

type DashboardConnectionUpsertRequest struct {
	Name       string `json:"name"`
	SystemName string `json:"system_name"`
	ServiceURL string `json:"service_url"`
	Client     string `json:"client"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	AccessMode string `json:"access_mode"`
}

type DashboardConnectionEditRequest struct {
	OldName string `json:"old_name"`
	DashboardConnectionUpsertRequest
}

type DashboardConnectionStatus struct {
	ActiveDefault    string `json:"active_default"`
	Connected        bool   `json:"connected"`
	Transport        string `json:"transport"`
	HTTPAddr         string `json:"http_addr,omitempty"`
	TotalConnections int    `json:"total_connections"`
}

type DashboardMutationResult struct {
	OK      bool                      `json:"ok"`
	Message string                    `json:"message,omitempty"`
	Error   string                    `json:"error,omitempty"`
	Status  DashboardConnectionStatus `json:"status"`
}

type DashboardDocumentationContext struct {
	Transport            string
	HTTPAddr             string
	MCPPath              string
	HealthPath           string
	DashboardPath        string
	DocumentationPath    string
	StatusPath           string
	ListPath             string
	ConnectPath          string
	DisconnectPath       string
	EditPath             string
	SwitchPath           string
	StateFile            string
	SupportsEmptyStartup bool
	ActiveConnection     string
	TotalConnections     int
}
