package dashboard

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/zmcp/odata-mcp/internal/bridge"
	"github.com/zmcp/odata-mcp/internal/config"
	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/models"
)

type Provider interface {
	Status(ctx context.Context) (*models.DashboardConnectionStatus, error)
	Connections(ctx context.Context) ([]models.DashboardConnection, error)
	Connect(ctx context.Context, req models.DashboardConnectionUpsertRequest) (*models.DashboardMutationResult, error)
	Disconnect(ctx context.Context, name string) (*models.DashboardMutationResult, error)
	Edit(ctx context.Context, req models.DashboardConnectionEditRequest) (*models.DashboardMutationResult, error)
	Switch(ctx context.Context, name string) (*models.DashboardMutationResult, error)
	DocsContext(ctx context.Context) (*models.DashboardDocumentationContext, error)
	RestoreActiveConnection(ctx context.Context) error
}

type bridgeController interface {
	ApplyConfig(cfg config.Config) error
	GetConfigSnapshot() (config.Config, error)
}

type BridgeProvider struct {
	bridge     bridgeController
	registry   *ConnectionRegistry
	baseConfig config.Config
	transport  string
	httpAddr   string
}

func NewBridgeProvider(odataBridge *bridge.ODataMCPBridge, baseConfig *config.Config, transportType, httpAddr string) *BridgeProvider {
	registry := NewConnectionRegistry(DefaultStateFile())
	registry.Load()

	cfgCopy := config.Config{}
	if baseConfig != nil {
		cfgCopy = *baseConfig
		if baseConfig.Cookies != nil {
			cfgCopy.Cookies = make(map[string]string, len(baseConfig.Cookies))
			for key, value := range baseConfig.Cookies {
				cfgCopy.Cookies[key] = value
			}
		}
		if baseConfig.AllowedEntities != nil {
			cfgCopy.AllowedEntities = append([]string(nil), baseConfig.AllowedEntities...)
		}
		if baseConfig.AllowedFunctions != nil {
			cfgCopy.AllowedFunctions = append([]string(nil), baseConfig.AllowedFunctions...)
		}
	}

	return &BridgeProvider{
		bridge:     odataBridge,
		registry:   registry,
		baseConfig: cfgCopy,
		transport:  transportType,
		httpAddr:   httpAddr,
	}
}

func (p *BridgeProvider) RestoreActiveConnection(ctx context.Context) error {
	active := p.registry.Active()
	if active == "" {
		p.rewriteConnections("", false)
		return p.bridge.ApplyConfig(p.configFor(ConnectionInfo{}))
	}

	conn, ok := p.registry.Get(active)
	if !ok {
		p.rewriteConnections("", false)
		return p.bridge.ApplyConfig(p.configFor(ConnectionInfo{}))
	}

	if err := p.applyConnection(ctx, conn); err != nil {
		p.rewriteConnections("", false)
		return err
	}

	p.rewriteConnections(conn.Name, true)
	return nil
}

func (p *BridgeProvider) Status(ctx context.Context) (*models.DashboardConnectionStatus, error) {
	_ = ctx
	conns := p.registry.ListAll()
	active := p.registry.Active()
	connected := false
	if active != "" {
		if conn, ok := p.registry.Get(active); ok {
			connected = conn.Connected
		}
	}

	return &models.DashboardConnectionStatus{
		ActiveDefault:    active,
		Connected:        connected,
		Transport:        p.transport,
		HTTPAddr:         p.httpAddr,
		TotalConnections: len(conns),
	}, nil
}

func (p *BridgeProvider) Connections(ctx context.Context) ([]models.DashboardConnection, error) {
	_ = ctx
	conns := p.registry.ListAll()
	result := make([]models.DashboardConnection, 0, len(conns))
	for _, conn := range conns {
		result = append(result, models.DashboardConnection{
			Name:           conn.Name,
			SystemName:     conn.SystemName,
			ServiceURL:     conn.ServiceURL,
			SafeServiceURL: conn.safeServiceURL(),
			Client:         conn.Client,
			Username:       conn.Username,
			AccessMode:     conn.AccessMode,
			Connected:      conn.Connected,
		})
	}
	return result, nil
}

func (p *BridgeProvider) Connect(ctx context.Context, req models.DashboardConnectionUpsertRequest) (*models.DashboardMutationResult, error) {
	conn, err := p.connectionFromRequest(req, true)
	if err != nil {
		return p.mutationError(ctx, err), nil
	}
	if _, exists := p.registry.Get(conn.Name); exists {
		return p.mutationError(ctx, fmt.Errorf("connection %q already exists", conn.Name)), nil
	}

	if err := p.applyConnection(ctx, conn); err != nil {
		return p.mutationError(ctx, err), nil
	}

	conns := p.registry.ListAll()
	conns = append(conns, conn)
	p.replaceConnections(conns, conn.Name, true)

	return p.mutationResult(ctx, "connected"), nil
}

func (p *BridgeProvider) Disconnect(ctx context.Context, name string) (*models.DashboardMutationResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return p.mutationError(ctx, fmt.Errorf("missing connection name")), nil
	}

	active := p.registry.Active()
	conns := p.registry.ListAll()
	filtered := make([]ConnectionInfo, 0, len(conns))
	for _, conn := range conns {
		if conn.Name != name {
			filtered = append(filtered, conn)
		}
	}
	if len(filtered) == len(conns) {
		return p.mutationError(ctx, fmt.Errorf("connection %q not found", name)), nil
	}

	nextActive := active
	if active == name {
		nextActive = ""
		if len(filtered) > 0 {
			nextActive = filtered[0].Name
		}
	}

	if nextActive == "" {
		if err := p.bridge.ApplyConfig(p.configFor(ConnectionInfo{})); err != nil {
			return p.mutationError(ctx, err), nil
		}
		p.replaceConnections(filtered, "", false)
		return p.mutationResult(ctx, "disconnected"), nil
	}

	nextConn, ok := findConnection(filtered, nextActive)
	if !ok {
		return p.mutationError(ctx, fmt.Errorf("connection %q not found", nextActive)), nil
	}
	if err := p.applyConnection(ctx, nextConn); err != nil {
		return p.mutationError(ctx, err), nil
	}

	p.replaceConnections(filtered, nextActive, true)
	return p.mutationResult(ctx, "disconnected"), nil
}

func (p *BridgeProvider) Edit(ctx context.Context, req models.DashboardConnectionEditRequest) (*models.DashboardMutationResult, error) {
	oldName := strings.TrimSpace(req.OldName)
	if oldName == "" {
		return p.mutationError(ctx, fmt.Errorf("missing old connection name")), nil
	}

	current, ok := p.registry.Get(oldName)
	if !ok {
		return p.mutationError(ctx, fmt.Errorf("connection %q not found", oldName)), nil
	}
	if strings.TrimSpace(req.Password) == "" {
		req.Password = current.Password
	}

	conn, err := p.connectionFromRequest(req.DashboardConnectionUpsertRequest, true)
	if err != nil {
		return p.mutationError(ctx, err), nil
	}

	if conn.Name != oldName {
		if _, exists := p.registry.Get(conn.Name); exists {
			return p.mutationError(ctx, fmt.Errorf("connection %q already exists", conn.Name)), nil
		}
	}

	active := p.registry.Active()
	if active == oldName {
		if err := p.applyConnection(ctx, conn); err != nil {
			return p.mutationError(ctx, err), nil
		}
	}

	conns := p.registry.ListAll()
	for i := range conns {
		if conns[i].Name == oldName {
			conns[i] = conn
		}
	}

	nextActive := active
	connected := active != ""
	if active == oldName {
		nextActive = conn.Name
		connected = true
	}
	p.replaceConnections(conns, nextActive, connected)

	return p.mutationResult(ctx, "saved"), nil
}

func (p *BridgeProvider) Switch(ctx context.Context, name string) (*models.DashboardMutationResult, error) {
	name = strings.TrimSpace(name)
	conn, ok := p.registry.Get(name)
	if !ok {
		return p.mutationError(ctx, fmt.Errorf("connection %q not found", name)), nil
	}

	if err := p.applyConnection(ctx, conn); err != nil {
		return p.mutationError(ctx, err), nil
	}

	p.rewriteConnections(name, true)
	return p.mutationResult(ctx, "switched"), nil
}

func (p *BridgeProvider) DocsContext(ctx context.Context) (*models.DashboardDocumentationContext, error) {
	status, err := p.Status(ctx)
	if err != nil {
		return nil, err
	}

	mcpPath := "/mcp"
	if p.transport == "http" || p.transport == "sse" {
		mcpPath = "/rpc"
	}

	return &models.DashboardDocumentationContext{
		Transport:            p.transport,
		HTTPAddr:             p.httpAddr,
		MCPPath:              mcpPath,
		HealthPath:           "/health",
		DashboardPath:        "/dashboard",
		DocumentationPath:    "/dashboard/docs",
		StatusPath:           "/api/status",
		ListPath:             "/api/databases",
		ConnectPath:          "/api/connect",
		DisconnectPath:       "/api/disconnect",
		EditPath:             "/api/edit",
		SwitchPath:           "/api/switch",
		StateFile:            p.registry.Path(),
		SupportsEmptyStartup: true,
		ActiveConnection:     status.ActiveDefault,
		TotalConnections:     status.TotalConnections,
	}, nil
}

func (p *BridgeProvider) mutationResult(ctx context.Context, message string) *models.DashboardMutationResult {
	status, err := p.Status(ctx)
	if err != nil {
		status = &models.DashboardConnectionStatus{
			Transport: p.transport,
			HTTPAddr:  p.httpAddr,
		}
	}
	return &models.DashboardMutationResult{
		OK:      true,
		Message: message,
		Status:  *status,
	}
}

func (p *BridgeProvider) mutationError(ctx context.Context, err error) *models.DashboardMutationResult {
	result := p.mutationResult(ctx, "")
	result.OK = false
	result.Error = err.Error()
	return result
}

func (p *BridgeProvider) applyConnection(ctx context.Context, conn ConnectionInfo) error {
	_ = ctx
	return p.bridge.ApplyConfig(p.configFor(conn))
}

func (p *BridgeProvider) configFor(conn ConnectionInfo) config.Config {
	cfg := p.baseConfig
	if p.baseConfig.Cookies != nil {
		cfg.Cookies = make(map[string]string, len(p.baseConfig.Cookies))
		for key, value := range p.baseConfig.Cookies {
			cfg.Cookies[key] = value
		}
	}
	if p.baseConfig.AllowedEntities != nil {
		cfg.AllowedEntities = append([]string(nil), p.baseConfig.AllowedEntities...)
	}
	if p.baseConfig.AllowedFunctions != nil {
		cfg.AllowedFunctions = append([]string(nil), p.baseConfig.AllowedFunctions...)
	}

	cfg.ServiceURL = ""
	cfg.Username = ""
	cfg.Password = ""
	cfg.CookieFile = ""
	cfg.CookieString = ""
	cfg.Cookies = nil
	cfg.ReadOnly = false
	cfg.ReadOnlyButFunctions = false

	if strings.TrimSpace(conn.Name) == "" {
		return cfg
	}

	cfg.ServiceURL = normalizeServiceURL(conn.ServiceURL, conn.Client)
	cfg.Username = conn.Username
	cfg.Password = conn.Password
	if strings.ToLower(conn.AccessMode) == "restricted" {
		cfg.ReadOnly = true
	}

	return cfg
}

func (p *BridgeProvider) replaceConnections(conns []ConnectionInfo, active string, connected bool) {
	for i := range conns {
		conns[i].Connected = connected && conns[i].Name == active
	}
	p.registry.ReplaceLoadedConnections(conns, active)
}

func (p *BridgeProvider) rewriteConnections(active string, connected bool) {
	p.replaceConnections(p.registry.ListAll(), active, connected)
}

func (p *BridgeProvider) connectionFromRequest(req models.DashboardConnectionUpsertRequest, requirePassword bool) (ConnectionInfo, error) {
	conn := ConnectionInfo{
		Name:       strings.TrimSpace(req.Name),
		SystemName: strings.TrimSpace(req.SystemName),
		ServiceURL: strings.TrimSpace(req.ServiceURL),
		Client:     strings.TrimSpace(req.Client),
		Username:   strings.TrimSpace(req.Username),
		Password:   req.Password,
		AccessMode: normalizeAccessMode(req.AccessMode),
	}

	switch {
	case conn.Name == "":
		return ConnectionInfo{}, fmt.Errorf("connection name is required")
	case conn.SystemName == "":
		return ConnectionInfo{}, fmt.Errorf("system name is required")
	case conn.ServiceURL == "":
		return ConnectionInfo{}, fmt.Errorf("service URL is required")
	case conn.Username == "":
		return ConnectionInfo{}, fmt.Errorf("username is required")
	case requirePassword && strings.TrimSpace(conn.Password) == "":
		return ConnectionInfo{}, fmt.Errorf("password is required")
	}

	if _, err := url.Parse(conn.ServiceURL); err != nil {
		return ConnectionInfo{}, fmt.Errorf("invalid service URL: %w", err)
	}

	return conn, nil
}

func normalizeAccessMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "restricted") {
		return "restricted"
	}
	return "unrestricted"
}

func normalizeServiceURL(rawURL, clientID string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	if strings.TrimSpace(clientID) != "" {
		query := parsed.Query()
		query.Set("sap-client", strings.TrimSpace(clientID))
		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}

func findConnection(conns []ConnectionInfo, name string) (ConnectionInfo, bool) {
	for _, conn := range conns {
		if conn.Name == name {
			return conn, true
		}
	}
	return ConnectionInfo{}, false
}

func CurrentProtocolVersion(cfg config.Config) string {
	if cfg.ProtocolVersion != "" {
		return cfg.ProtocolVersion
	}
	return constants.MCPProtocolVersion
}
