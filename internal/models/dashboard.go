package models

import "time"

type DashboardService struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ServiceURL     string `json:"service_url"`
	SafeServiceURL string `json:"safe_service_url"`
}

type DashboardOperation struct {
	ID          string `json:"id"`
	Verb        string `json:"verb"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name,omitempty"`
	EntitySet   string `json:"entity_set"`
	ToolName    string `json:"tool_name,omitempty"`
	Mode        string `json:"mode"`
	Enabled     bool   `json:"enabled"`
}

type DashboardEntity struct {
	ID          string               `json:"id"`
	Label       string               `json:"label"`
	Description string               `json:"description,omitempty"`
	Operations  []DashboardOperation `json:"operations"`
}

type DashboardSystem struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	BaseURL     string             `json:"base_url,omitempty"`
	Client      string             `json:"client,omitempty"`
	Username    string             `json:"username"`
	HasPassword bool               `json:"has_password"`
	AccessMode  string             `json:"access_mode"`
	Connected   bool               `json:"connected"`
	Active      bool               `json:"active"`
	Services    []DashboardService `json:"services"`
	Entities    []DashboardEntity  `json:"entities"`
	ServiceNote string             `json:"service_note,omitempty"`
}

type DashboardSystemUpsertRequest struct {
	OldID      string `json:"old_id,omitempty"`
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	Client     string `json:"client"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	AccessMode string `json:"access_mode"`
}

type DashboardServiceUpsertRequest struct {
	SystemID   string `json:"system_id"`
	OldID      string `json:"old_id,omitempty"`
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	ServiceURL string `json:"service_url"`
}

type DashboardEntityUpsertRequest struct {
	SystemID     string `json:"system_id"`
	OldID        string `json:"old_id,omitempty"`
	ID           string `json:"id,omitempty"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	GenerateCRUD bool   `json:"generate_crud,omitempty"`
}

type DashboardOperationUpsertRequest struct {
	SystemID  string `json:"system_id"`
	EntityID  string `json:"entity_id"`
	OldID     string `json:"old_id,omitempty"`
	ID        string `json:"id,omitempty"`
	Verb      string `json:"verb"`
	ServiceID string `json:"service_id"`
	EntitySet string `json:"entity_set"`
	Mode      string `json:"mode"`
	Enabled   bool   `json:"enabled"`
}

type DashboardDeleteRequest struct {
	SystemID    string `json:"system_id,omitempty"`
	EntityID    string `json:"entity_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	ServiceID   string `json:"service_id,omitempty"`
	ID          string `json:"id,omitempty"`
}

type DashboardActivationRequest struct {
	SystemID string `json:"system_id"`
}

type DashboardHierarchyStatus struct {
	ActiveSystemID   string `json:"active_system_id"`
	ActiveSystemName string `json:"active_system_name,omitempty"`
	Connected        bool   `json:"connected"`
	Transport        string `json:"transport"`
	HTTPAddr         string `json:"http_addr,omitempty"`
	TotalSystems     int    `json:"total_systems"`
	TotalServices    int    `json:"total_services"`
	TotalEntities    int    `json:"total_entities"`
	TotalOperations  int    `json:"total_operations"`
}

type DashboardMutationResult struct {
	OK      bool                     `json:"ok"`
	Message string                   `json:"message,omitempty"`
	Error   string                   `json:"error,omitempty"`
	Status  DashboardHierarchyStatus `json:"status"`
}

type DashboardServiceDiscovery struct {
	SystemID        string                `json:"system_id"`
	ServiceID       string                `json:"service_id"`
	ServiceName     string                `json:"service_name"`
	ServiceURL      string                `json:"service_url"`
	SafeServiceURL  string                `json:"safe_service_url"`
	Version         string                `json:"version"`
	SchemaNamespace string                `json:"schema_namespace"`
	ContainerName   string                `json:"container_name"`
	EntitySets      []EntitySetSummary    `json:"entity_sets"`
	FunctionImports []FunctionImportBrief `json:"function_imports"`
}

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

type DashboardServiceTestResult struct {
	ServiceID       string          `json:"service_id"`
	ServiceName     string          `json:"service_name"`
	ServiceURL      string          `json:"service_url"`
	OK              bool            `json:"ok"`
	Message         string          `json:"message"`
	DurationMs      int64           `json:"duration_ms"`
	IsSAPService    bool            `json:"is_sap_service"`
	MetadataSummary MetadataSummary `json:"metadata_summary"`
	Version         string          `json:"version"`
}

type DashboardSystemTestResult struct {
	OK         bool                         `json:"ok"`
	Message    string                       `json:"message"`
	SystemID   string                       `json:"system_id"`
	SystemName string                       `json:"system_name"`
	DurationMs int64                        `json:"duration_ms"`
	Services   []DashboardServiceTestResult `json:"services"`
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

type DashboardDocumentationContext struct {
	Transport            string
	HTTPAddr             string
	MCPPath              string
	HealthPath           string
	DashboardPath        string
	DocumentationPath    string
	StatusPath           string
	SystemsPath          string
	SaveSystemPath       string
	DeleteSystemPath     string
	ActivateSystemPath   string
	SaveServicePath      string
	DeleteServicePath    string
	SaveEntityPath       string
	DeleteEntityPath     string
	SaveOperationPath    string
	DeleteOperationPath  string
	DiscoveryPath        string
	StateFile            string
	SupportsEmptyStartup bool
	ActiveSystem         string
	TotalSystems         int
	TotalEntities        int
	TotalOperations      int
}
