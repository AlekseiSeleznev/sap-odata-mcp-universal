package dashboard

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/bridge"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/config"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
)

type Provider interface {
	Status(ctx context.Context) (*models.DashboardHierarchyStatus, error)
	Systems(ctx context.Context) ([]models.DashboardSystem, error)
	TestSystem(ctx context.Context, id string) (*models.DashboardSystemTestResult, error)
	SaveSystem(ctx context.Context, req models.DashboardSystemUpsertRequest) (*models.DashboardMutationResult, error)
	DeleteSystem(ctx context.Context, id string) (*models.DashboardMutationResult, error)
	ActivateSystem(ctx context.Context, id string) (*models.DashboardMutationResult, error)
	SaveService(ctx context.Context, req models.DashboardServiceUpsertRequest) (*models.DashboardMutationResult, error)
	DeleteService(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error)
	SaveEntity(ctx context.Context, req models.DashboardEntityUpsertRequest) (*models.DashboardMutationResult, error)
	DeleteEntity(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error)
	SaveOperation(ctx context.Context, req models.DashboardOperationUpsertRequest) (*models.DashboardMutationResult, error)
	DeleteOperation(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error)
	DiscoverService(ctx context.Context, systemID, serviceID string) (*models.DashboardServiceDiscovery, error)
	DocsContext(ctx context.Context) (*models.DashboardDocumentationContext, error)
	RestoreActiveConnection(ctx context.Context) error
}

type BridgeProvider struct {
	registry   *Registry
	runtime    *HierarchicalRuntime
	transport  string
	httpAddr   string
	baseConfig config.Config
}

func NewBridgeProvider(odataBridge *bridge.ODataMCPBridge, baseConfig *config.Config, transportType, httpAddr string) *BridgeProvider {
	cfgCopy := config.Config{}
	if baseConfig != nil {
		cfgCopy = *baseConfig
	}

	registry := NewConnectionRegistry(DefaultStateFile())
	registry.Load()

	return &BridgeProvider{
		registry:   registry,
		runtime:    NewHierarchicalRuntime(odataBridge, cfgCopy),
		transport:  transportType,
		httpAddr:   httpAddr,
		baseConfig: cfgCopy,
	}
}

func (p *BridgeProvider) RestoreActiveConnection(ctx context.Context) error {
	_ = ctx
	systems := p.registry.ListAll()
	active := p.registry.Active()
	return p.applyAndPersist(ctx, systems, active)
}

func (p *BridgeProvider) Status(ctx context.Context) (*models.DashboardHierarchyStatus, error) {
	_ = ctx
	systems := p.registry.ListAll()
	activeID := p.registry.Active()
	status := &models.DashboardHierarchyStatus{
		ActiveSystemID: activeID,
		Connected:      false,
		Transport:      p.transport,
		HTTPAddr:       p.httpAddr,
		TotalSystems:   len(systems),
	}
	for _, system := range systems {
		if system.ID == activeID {
			status.ActiveSystemName = system.Name
			status.Connected = system.Connected
		}
		status.TotalServices += len(system.Services)
		status.TotalEntities += len(system.Entities)
		for _, entity := range system.Entities {
			status.TotalOperations += len(entity.Operations)
		}
	}
	return status, nil
}

func (p *BridgeProvider) Systems(ctx context.Context) ([]models.DashboardSystem, error) {
	_ = ctx
	systems := p.registry.ListAll()
	activeID := p.registry.Active()
	result := make([]models.DashboardSystem, 0, len(systems))
	for _, system := range systems {
		item := models.DashboardSystem{
			ID:          system.ID,
			Name:        system.Name,
			BaseURL:     system.BaseURL,
			Client:      system.Client,
			Username:    system.Username,
			HasPassword: system.Password != "",
			AccessMode:  system.AccessMode,
			Connected:   system.Connected,
			Active:      system.ID == activeID,
			Services:    make([]models.DashboardService, 0, len(system.Services)),
			Entities:    make([]models.DashboardEntity, 0, len(system.Entities)),
		}
		for _, service := range system.Services {
			item.Services = append(item.Services, models.DashboardService{
				ID:             service.ID,
				Name:           service.Name,
				ServiceURL:     service.ServiceURL,
				SafeServiceURL: service.safeServiceURL(),
			})
		}
		if len(item.Services) == 0 {
			item.ServiceNote = "No services added yet"
		}
		for _, entity := range system.Entities {
			entityItem := models.DashboardEntity{
				ID:          entity.ID,
				Label:       entity.Label,
				Description: entity.Description,
				Operations:  make([]models.DashboardOperation, 0, len(entity.Operations)),
			}
			for _, op := range entity.Operations {
				service, _ := findService(system.Services, op.ServiceID)
				entityItem.Operations = append(entityItem.Operations, models.DashboardOperation{
					ID:          op.ID,
					Name:        operationDisplayName(op),
					Verb:        normalizeVerb(op.Verb),
					ServiceID:   op.ServiceID,
					ServiceName: service.Name,
					EntitySet:   op.EntitySet,
					Query:       copyQueryOptions(op.Query),
					ToolName:    p.runtime.ToolName(entity.ID, op.ID),
					Mode:        normalizeMode(op.Mode),
					Enabled:     op.Enabled,
				})
			}
			item.Entities = append(item.Entities, entityItem)
		}
		result = append(result, item)
	}
	return result, nil
}

func (p *BridgeProvider) SaveSystem(ctx context.Context, req models.DashboardSystemUpsertRequest) (*models.DashboardMutationResult, error) {
	systems := p.registry.ListAll()
	oldID := strings.TrimSpace(req.OldID)
	current, oldIndex := findSystemIndex(systems, oldID)
	if oldID != "" && oldIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", oldID)), nil
	}

	if strings.TrimSpace(req.Password) == "" && oldIndex >= 0 {
		req.Password = current.Password
	}

	system, err := p.systemFromRequest(systems, req, current)
	if err != nil {
		return p.mutationError(ctx, err), nil
	}

	active := p.registry.Active()
	if oldIndex >= 0 {
		system.Services = current.Services
		system.Entities = current.Entities
		systems[oldIndex] = system
		if active == oldID {
			active = system.ID
		}
	} else {
		systems = append(systems, system)
		if active == "" {
			active = system.ID
		}
	}

	if oldIndex >= 0 && active == system.ID && onlyAccessModeChanged(current, system) {
		for i := range systems {
			systems[i].Connected = active != "" && systems[i].ID == active
		}
		p.registry.ReplaceLoadedSystems(systems, active)
		p.runtime.SetActiveAccessMode(system.ID, system.AccessMode)
		return p.mutationResult(ctx, "saved"), nil
	}

	if err := p.applyAndPersist(ctx, systems, active); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "saved"), nil
}

func (p *BridgeProvider) TestSystem(ctx context.Context, id string) (*models.DashboardSystemTestResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("system id is required")
	}

	system, ok := p.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("system %q not found", id)
	}
	if len(system.Services) == 0 {
		return &models.DashboardSystemTestResult{
			OK:         false,
			Message:    "system has no services to test",
			SystemID:   system.ID,
			SystemName: system.Name,
			Services:   []models.DashboardServiceTestResult{},
		}, nil
	}

	startedAll := time.Now()
	result := &models.DashboardSystemTestResult{
		OK:         true,
		SystemID:   system.ID,
		SystemName: system.Name,
		Services:   make([]models.DashboardServiceTestResult, 0, len(system.Services)),
	}
	for _, service := range system.Services {
		started := time.Now()
		discovery, err := p.runtime.Discover(ctx, system, service.ID)
		duration := time.Since(started).Milliseconds()
		item := models.DashboardServiceTestResult{
			ServiceID:   service.ID,
			ServiceName: service.Name,
			ServiceURL:  service.safeServiceURL(),
			DurationMs:  duration,
		}
		if err != nil {
			item.OK = false
			item.Message = err.Error()
			result.OK = false
		} else {
			item.OK = true
			item.Message = "metadata loaded"
			item.ServiceURL = discovery.SafeServiceURL
			item.IsSAPService = strings.HasPrefix(strings.ToLower(discovery.SchemaNamespace), "sap") ||
				strings.Contains(strings.ToLower(discovery.ServiceURL), "/sap/opu/odata/")
			item.Version = discovery.Version
			item.MetadataSummary = models.MetadataSummary{
				EntitySets:      len(discovery.EntitySets),
				FunctionImports: len(discovery.FunctionImports),
			}
		}
		result.Services = append(result.Services, item)
	}
	result.DurationMs = time.Since(startedAll).Milliseconds()
	okCount := 0
	for _, service := range result.Services {
		if service.OK {
			okCount++
		}
	}
	if result.OK {
		result.Message = fmt.Sprintf("connection to %s verified (%d/%d services)", system.Name, okCount, len(result.Services))
	} else {
		result.Message = fmt.Sprintf("connection test failed for %s (%d/%d services)", system.Name, okCount, len(result.Services))
	}
	return result, nil
}

func (p *BridgeProvider) DeleteSystem(ctx context.Context, id string) (*models.DashboardMutationResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return p.mutationError(ctx, fmt.Errorf("system id is required")), nil
	}

	systems := p.registry.ListAll()
	filtered := make([]SystemInfo, 0, len(systems))
	for _, system := range systems {
		if system.ID != id {
			filtered = append(filtered, system)
		}
	}
	if len(filtered) == len(systems) {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", id)), nil
	}

	active := p.registry.Active()
	if active == id {
		active = ""
		if len(filtered) > 0 {
			active = filtered[0].ID
		}
	}
	if err := p.applyAndPersist(ctx, filtered, active); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "deleted"), nil
}

func (p *BridgeProvider) ActivateSystem(ctx context.Context, id string) (*models.DashboardMutationResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return p.mutationError(ctx, fmt.Errorf("system id is required")), nil
	}
	systems := p.registry.ListAll()
	if _, idx := findSystemIndex(systems, id); idx < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", id)), nil
	}
	if err := p.applyAndPersist(ctx, systems, id); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "activated"), nil
}

func (p *BridgeProvider) SaveService(ctx context.Context, req models.DashboardServiceUpsertRequest) (*models.DashboardMutationResult, error) {
	systems := p.registry.ListAll()
	system, sysIndex := findSystemIndex(systems, req.SystemID)
	if sysIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", req.SystemID)), nil
	}

	service, err := p.serviceFromRequest(system, req)
	if err != nil {
		return p.mutationError(ctx, err), nil
	}

	oldID := strings.TrimSpace(req.OldID)
	if oldID != "" {
		replaced := false
		for i := range system.Services {
			if system.Services[i].ID == oldID {
				system.Services[i] = service
				replaced = true
				break
			}
		}
		if !replaced {
			return p.mutationError(ctx, fmt.Errorf("service %q not found", oldID)), nil
		}
		for i := range system.Entities {
			for j := range system.Entities[i].Operations {
				if system.Entities[i].Operations[j].ServiceID == oldID {
					system.Entities[i].Operations[j].ServiceID = service.ID
				}
			}
		}
	} else {
		system.Services = append(system.Services, service)
	}
	systems[sysIndex] = sanitizeSystem(system)

	if err := p.applyAndPersist(ctx, systems, p.registry.Active()); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "saved"), nil
}

func (p *BridgeProvider) DeleteService(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error) {
	systems := p.registry.ListAll()
	system, sysIndex := findSystemIndex(systems, req.SystemID)
	if sysIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", req.SystemID)), nil
	}

	serviceID := strings.TrimSpace(req.ServiceID)
	if serviceID == "" {
		serviceID = strings.TrimSpace(req.ID)
	}
	if serviceID == "" {
		return p.mutationError(ctx, fmt.Errorf("service id is required")), nil
	}

	usedBy := make([]string, 0)
	for _, entity := range system.Entities {
		for _, op := range entity.Operations {
			if op.ServiceID == serviceID {
				usedBy = append(usedBy, entity.Label+"."+normalizeVerb(op.Verb))
			}
		}
	}
	if len(usedBy) > 0 {
		return p.mutationError(ctx, fmt.Errorf("service is still used by operations: %s", strings.Join(usedBy, ", "))), nil
	}

	filtered := make([]ServiceInfo, 0, len(system.Services))
	for _, service := range system.Services {
		if service.ID != serviceID {
			filtered = append(filtered, service)
		}
	}
	if len(filtered) == len(system.Services) {
		return p.mutationError(ctx, fmt.Errorf("service %q not found", serviceID)), nil
	}
	system.Services = filtered
	systems[sysIndex] = sanitizeSystem(system)

	if err := p.applyAndPersist(ctx, systems, p.registry.Active()); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "deleted"), nil
}

func (p *BridgeProvider) SaveEntity(ctx context.Context, req models.DashboardEntityUpsertRequest) (*models.DashboardMutationResult, error) {
	systems := p.registry.ListAll()
	system, sysIndex := findSystemIndex(systems, req.SystemID)
	if sysIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", req.SystemID)), nil
	}

	entity, err := entityFromRequest(system, req)
	if err != nil {
		return p.mutationError(ctx, err), nil
	}

	oldID := strings.TrimSpace(req.OldID)
	if oldID != "" {
		replaced := false
		for i := range system.Entities {
			if system.Entities[i].ID == oldID {
				entity.Operations = system.Entities[i].Operations
				system.Entities[i] = entity
				replaced = true
				break
			}
		}
		if !replaced {
			return p.mutationError(ctx, fmt.Errorf("object %q not found", oldID)), nil
		}
	} else {
		system.Entities = append(system.Entities, entity)
	}
	systems[sysIndex] = sanitizeSystem(system)

	if err := p.applyAndPersist(ctx, systems, p.registry.Active()); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "saved"), nil
}

func (p *BridgeProvider) DeleteEntity(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error) {
	systems := p.registry.ListAll()
	system, sysIndex := findSystemIndex(systems, req.SystemID)
	if sysIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", req.SystemID)), nil
	}

	entityID := strings.TrimSpace(req.EntityID)
	if entityID == "" {
		entityID = strings.TrimSpace(req.ID)
	}
	if entityID == "" {
		return p.mutationError(ctx, fmt.Errorf("object id is required")), nil
	}

	filtered := make([]EntityInfo, 0, len(system.Entities))
	for _, entity := range system.Entities {
		if entity.ID != entityID {
			filtered = append(filtered, entity)
		}
	}
	if len(filtered) == len(system.Entities) {
		return p.mutationError(ctx, fmt.Errorf("object %q not found", entityID)), nil
	}
	system.Entities = filtered
	systems[sysIndex] = sanitizeSystem(system)

	if err := p.applyAndPersist(ctx, systems, p.registry.Active()); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "deleted"), nil
}

func (p *BridgeProvider) SaveOperation(ctx context.Context, req models.DashboardOperationUpsertRequest) (*models.DashboardMutationResult, error) {
	systems := p.registry.ListAll()
	system, sysIndex := findSystemIndex(systems, req.SystemID)
	if sysIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", req.SystemID)), nil
	}
	entity, entityIndex := findEntityIndex(system.Entities, req.EntityID)
	if entityIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("object %q not found", req.EntityID)), nil
	}

	op, err := p.operationFromRequest(ctx, system, entity, req)
	if err != nil {
		return p.mutationError(ctx, err), nil
	}

	oldID := strings.TrimSpace(req.OldID)
	if oldID != "" {
		replaced := false
		for i := range entity.Operations {
			if entity.Operations[i].ID == oldID {
				entity.Operations[i] = op
				replaced = true
				break
			}
		}
		if !replaced {
			return p.mutationError(ctx, fmt.Errorf("operation %q not found", oldID)), nil
		}
	} else {
		entity.Operations = append(entity.Operations, op)
	}
	system.Entities[entityIndex] = entity
	systems[sysIndex] = sanitizeSystem(system)

	if err := p.applyAndPersist(ctx, systems, p.registry.Active()); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "saved"), nil
}

func (p *BridgeProvider) DeleteOperation(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error) {
	systems := p.registry.ListAll()
	system, sysIndex := findSystemIndex(systems, req.SystemID)
	if sysIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("system %q not found", req.SystemID)), nil
	}
	entity, entityIndex := findEntityIndex(system.Entities, req.EntityID)
	if entityIndex < 0 {
		return p.mutationError(ctx, fmt.Errorf("object %q not found", req.EntityID)), nil
	}

	opID := strings.TrimSpace(req.OperationID)
	if opID == "" {
		opID = strings.TrimSpace(req.ID)
	}
	if opID == "" {
		return p.mutationError(ctx, fmt.Errorf("operation id is required")), nil
	}

	filtered := make([]OperationInfo, 0, len(entity.Operations))
	for _, op := range entity.Operations {
		if op.ID != opID {
			filtered = append(filtered, op)
		}
	}
	if len(filtered) == len(entity.Operations) {
		return p.mutationError(ctx, fmt.Errorf("operation %q not found", opID)), nil
	}
	entity.Operations = filtered
	system.Entities[entityIndex] = entity
	systems[sysIndex] = sanitizeSystem(system)

	if err := p.applyAndPersist(ctx, systems, p.registry.Active()); err != nil {
		return p.mutationError(ctx, err), nil
	}
	return p.mutationResult(ctx, "deleted"), nil
}

func (p *BridgeProvider) DiscoverService(ctx context.Context, systemID, serviceID string) (*models.DashboardServiceDiscovery, error) {
	_ = ctx
	system, ok := p.registry.Get(systemID)
	if !ok {
		return nil, fmt.Errorf("system %q not found", systemID)
	}
	return p.runtime.Discover(ctx, system, serviceID)
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
		SystemsPath:          "/api/systems",
		SaveSystemPath:       "/api/system/save",
		DeleteSystemPath:     "/api/system/delete",
		ActivateSystemPath:   "/api/system/activate",
		SaveServicePath:      "/api/service/save",
		DeleteServicePath:    "/api/service/delete",
		SaveEntityPath:       "/api/entity/save",
		DeleteEntityPath:     "/api/entity/delete",
		SaveOperationPath:    "/api/operation/save",
		DeleteOperationPath:  "/api/operation/delete",
		DiscoveryPath:        "/api/service/discover",
		StateFile:            p.registry.Path(),
		SupportsEmptyStartup: true,
		ActiveSystem:         status.ActiveSystemID,
		TotalSystems:         status.TotalSystems,
		TotalEntities:        status.TotalEntities,
		TotalOperations:      status.TotalOperations,
	}, nil
}

func (p *BridgeProvider) applyAndPersist(ctx context.Context, systems []SystemInfo, active string) error {
	var err error
	if active == "" {
		err = p.runtime.Clear()
	} else {
		system, idx := findSystemIndex(systems, active)
		if idx < 0 {
			return fmt.Errorf("active system %q not found", active)
		}
		err = p.runtime.ApplySystem(ctx, system)
	}
	if err != nil {
		return err
	}

	for i := range systems {
		systems[i].Connected = active != "" && systems[i].ID == active
	}
	p.registry.ReplaceLoadedSystems(systems, active)
	return nil
}

func (p *BridgeProvider) mutationResult(ctx context.Context, message string) *models.DashboardMutationResult {
	status, err := p.Status(ctx)
	if err != nil {
		status = &models.DashboardHierarchyStatus{
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

func (p *BridgeProvider) systemFromRequest(existing []SystemInfo, req models.DashboardSystemUpsertRequest, current SystemInfo) (SystemInfo, error) {
	system := SystemInfo{
		ID:         strings.TrimSpace(req.ID),
		Name:       strings.TrimSpace(req.Name),
		BaseURL:    strings.TrimSpace(req.BaseURL),
		Client:     strings.TrimSpace(req.Client),
		Username:   strings.TrimSpace(req.Username),
		Password:   req.Password,
		AccessMode: normalizeAccessMode(req.AccessMode),
		Services:   current.Services,
		Entities:   current.Entities,
	}
	switch {
	case system.Name == "":
		return SystemInfo{}, fmt.Errorf("system name is required")
	case system.Username == "":
		return SystemInfo{}, fmt.Errorf("username is required")
	case strings.TrimSpace(system.Password) == "" && strings.TrimSpace(req.OldID) == "":
		return SystemInfo{}, fmt.Errorf("password is required")
	}

	if system.BaseURL != "" {
		if _, err := url.Parse(system.BaseURL); err != nil {
			return SystemInfo{}, fmt.Errorf("invalid base URL: %w", err)
		}
	}

	if system.ID == "" && strings.TrimSpace(req.OldID) != "" && current.ID != "" {
		system.ID = current.ID
	}

	if system.ID == "" {
		system.ID = ensureUniqueID(system.Name+"-"+system.Client, func(id string) bool {
			for _, item := range existing {
				if item.ID == id {
					return true
				}
			}
			return false
		})
	}

	for _, item := range existing {
		if item.ID == req.OldID {
			continue
		}
		if item.ID == system.ID {
			return SystemInfo{}, fmt.Errorf("system id %q already exists", system.ID)
		}
	}

	return sanitizeSystem(system), nil
}

func onlyAccessModeChanged(before, after SystemInfo) bool {
	left := sanitizeSystem(before)
	right := sanitizeSystem(after)
	left.Connected = false
	right.Connected = false
	left.AccessMode = ""
	right.AccessMode = ""
	return reflect.DeepEqual(left, right)
}

func (p *BridgeProvider) serviceFromRequest(system SystemInfo, req models.DashboardServiceUpsertRequest) (ServiceInfo, error) {
	service := ServiceInfo{
		ID:         strings.TrimSpace(req.ID),
		Name:       strings.TrimSpace(req.Name),
		ServiceURL: strings.TrimSpace(req.ServiceURL),
	}
	switch {
	case service.Name == "":
		return ServiceInfo{}, fmt.Errorf("service name is required")
	case service.ServiceURL == "":
		return ServiceInfo{}, fmt.Errorf("service URL is required")
	}
	if _, err := url.Parse(service.ServiceURL); err != nil {
		return ServiceInfo{}, fmt.Errorf("invalid service URL: %w", err)
	}
	if service.ID == "" {
		service.ID = ensureUniqueID(service.Name, func(id string) bool {
			for _, item := range system.Services {
				if item.ID == id {
					return true
				}
			}
			return false
		})
	}
	for _, item := range system.Services {
		if item.ID == req.OldID {
			continue
		}
		if item.ID == service.ID {
			return ServiceInfo{}, fmt.Errorf("service id %q already exists", service.ID)
		}
	}
	return service, nil
}

func entityFromRequest(system SystemInfo, req models.DashboardEntityUpsertRequest) (EntityInfo, error) {
	entity := EntityInfo{
		ID:          strings.TrimSpace(req.ID),
		Label:       strings.TrimSpace(req.Label),
		Description: strings.TrimSpace(req.Description),
	}
	if entity.Label == "" {
		return EntityInfo{}, fmt.Errorf("object name is required")
	}
	if entity.ID == "" {
		entity.ID = ensureUniqueID(entity.Label, func(id string) bool {
			for _, item := range system.Entities {
				if item.ID == id {
					return true
				}
			}
			return false
		})
	}
	for _, item := range system.Entities {
		if item.ID == req.OldID {
			continue
		}
		if item.ID == entity.ID {
			return EntityInfo{}, fmt.Errorf("object id %q already exists", entity.ID)
		}
	}
	return entity, nil
}

func (p *BridgeProvider) operationFromRequest(ctx context.Context, system SystemInfo, entity EntityInfo, req models.DashboardOperationUpsertRequest) (OperationInfo, error) {
	op := OperationInfo{
		ID:        strings.TrimSpace(req.ID),
		Name:      strings.TrimSpace(req.Name),
		Verb:      normalizeVerb(req.Verb),
		ServiceID: strings.TrimSpace(req.ServiceID),
		EntitySet: strings.TrimSpace(req.EntitySet),
		Query:     normalizeOperationQuery(req.Query),
		Mode:      normalizeMode(req.Mode),
		Enabled:   req.Enabled,
	}
	if req.OldID == "" && !req.Enabled {
		op.Enabled = true
	}
	switch {
	case op.Verb == "":
		return OperationInfo{}, fmt.Errorf("operation verb is required")
	case op.ServiceID == "":
		return OperationInfo{}, fmt.Errorf("service is required")
	case op.EntitySet == "":
		return OperationInfo{}, fmt.Errorf("entity set is required")
	}
	if _, ok := findService(system.Services, op.ServiceID); !ok {
		return OperationInfo{}, fmt.Errorf("service %q not found", op.ServiceID)
	}
	if op.Name == "" {
		op.Name = defaultOperationName(op)
	}
	if op.ID == "" {
		idSource := entity.ID + "-" + op.Verb + "-" + op.EntitySet
		if strings.TrimSpace(req.Name) != "" {
			idSource = entity.ID + "-" + req.Name
		}
		op.ID = ensureUniqueID(idSource, func(id string) bool {
			for _, item := range entity.Operations {
				if item.ID == id {
					return true
				}
			}
			return false
		})
	}
	for _, item := range entity.Operations {
		if item.ID == req.OldID {
			continue
		}
		if item.ID == op.ID {
			return OperationInfo{}, fmt.Errorf("operation id %q already exists", op.ID)
		}
		if sameOperationBinding(item, op) {
			return OperationInfo{}, fmt.Errorf("operation binding %q/%s already exists for this object", op.Verb, op.EntitySet)
		}
	}

	discovery, err := p.runtime.Discover(ctx, system, op.ServiceID)
	if err != nil {
		return OperationInfo{}, err
	}
	foundEntitySet := false
	for _, entitySet := range discovery.EntitySets {
		if entitySet.Name == op.EntitySet {
			foundEntitySet = true
			break
		}
	}
	if !foundEntitySet {
		return OperationInfo{}, fmt.Errorf("entity set %q not found in selected service", op.EntitySet)
	}
	return op, nil
}

func defaultOperationName(op OperationInfo) string {
	verb := strings.ToUpper(normalizeVerb(op.Verb))
	if verb == "LIST" {
		verb = "GET list"
	}
	if strings.TrimSpace(op.EntitySet) == "" {
		return verb
	}
	return verb + " " + op.EntitySet
}

func operationDisplayName(op OperationInfo) string {
	if strings.TrimSpace(op.Name) != "" {
		return strings.TrimSpace(op.Name)
	}
	return defaultOperationName(op)
}

func normalizeOperationQuery(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	allowed := map[string]string{
		"$expand":  "$expand",
		"expand":   "$expand",
		"$select":  "$select",
		"select":   "$select",
		"$filter":  "$filter",
		"filter":   "$filter",
		"$orderby": "$orderby",
		"orderby":  "$orderby",
		"$top":     "$top",
		"top":      "$top",
		"$skip":    "$skip",
		"skip":     "$skip",
		"$count":   "$count",
		"count":    "$count",
	}
	query := make(map[string]string)
	for key, value := range raw {
		normalized, ok := allowed[strings.ToLower(strings.TrimSpace(key))]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		query[normalized] = value
	}
	if len(query) == 0 {
		return nil
	}
	return query
}

func copyQueryOptions(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	copy := make(map[string]string, len(raw))
	for key, value := range raw {
		copy[key] = value
	}
	return copy
}

func sameOperationBinding(left, right OperationInfo) bool {
	return normalizeVerb(left.Verb) == normalizeVerb(right.Verb) &&
		left.ServiceID == right.ServiceID &&
		left.EntitySet == right.EntitySet &&
		reflect.DeepEqual(normalizeOperationQuery(left.Query), normalizeOperationQuery(right.Query))
}

func findSystemIndex(systems []SystemInfo, id string) (SystemInfo, int) {
	for i, system := range systems {
		if system.ID == id {
			return system, i
		}
	}
	return SystemInfo{}, -1
}

func findEntityIndex(entities []EntityInfo, id string) (EntityInfo, int) {
	for i, entity := range entities {
		if entity.ID == id {
			return entity, i
		}
	}
	return EntityInfo{}, -1
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
