package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/bridge"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/client"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/config"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/constants"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/mcp"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/utils"
)

type runtimeBridge interface {
	ApplyConfig(cfg config.Config) error
	GetServer() *mcp.Server
}

type HierarchicalRuntime struct {
	bridge           runtimeBridge
	baseConfig       config.Config
	registered       []string
	clientCache      map[string]*client.ODataClient
	metadataCache    map[string]*models.ODataMetadata
	serviceCacheKeys map[string]string
	toolNames        map[string]string
	activeSystem     string
	activeAccess     string
	mu               sync.Mutex
}

func NewHierarchicalRuntime(odataBridge *bridge.ODataMCPBridge, baseConfig config.Config) *HierarchicalRuntime {
	runtime := &HierarchicalRuntime{
		bridge:           odataBridge,
		baseConfig:       baseConfig,
		clientCache:      make(map[string]*client.ODataClient),
		metadataCache:    make(map[string]*models.ODataMetadata),
		serviceCacheKeys: make(map[string]string),
		toolNames:        make(map[string]string),
	}
	runtime.registerCachedCSDLSummaryTool()
	runtime.registerWSDLBundleFetchTool()
	return runtime
}

func (r *HierarchicalRuntime) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.bridge.ApplyConfig(config.Config{}); err != nil {
		return err
	}
	server := r.bridge.GetServer()
	for _, name := range r.registered {
		server.RemoveTool(name)
	}
	r.registered = nil
	r.toolNames = make(map[string]string)
	r.clientCache = make(map[string]*client.ODataClient)
	r.metadataCache = make(map[string]*models.ODataMetadata)
	r.serviceCacheKeys = make(map[string]string)
	r.activeSystem = ""
	r.activeAccess = ""
	return nil
}

func (r *HierarchicalRuntime) ApplySystem(ctx context.Context, system SystemInfo) error {
	r.mu.Lock()
	if err := r.bridge.ApplyConfig(config.Config{}); err != nil {
		r.mu.Unlock()
		return err
	}

	server := r.bridge.GetServer()
	for _, name := range r.registered {
		server.RemoveTool(name)
	}
	r.registered = nil
	r.toolNames = make(map[string]string)
	r.clientCache = make(map[string]*client.ODataClient)
	r.metadataCache = make(map[string]*models.ODataMetadata)
	r.serviceCacheKeys = make(map[string]string)
	r.activeSystem = system.ID
	r.activeAccess = normalizeAccessMode(system.AccessMode)
	r.mu.Unlock()

	usedNames := make(map[string]struct{})
	for _, entity := range system.Entities {
		for _, op := range entity.Operations {
			if !op.Enabled {
				continue
			}

			service, ok := findService(system.Services, op.ServiceID)
			if !ok {
				continue
			}

			meta, err := r.metadataFor(ctx, system, service)
			if err != nil {
				return err
			}

			entitySet, ok := meta.EntitySets[op.EntitySet]
			if !ok {
				return fmt.Errorf("entity set %q not found in service %q", op.EntitySet, service.Name)
			}

			entityType := lookupEntityType(meta, entitySet.EntityType)
			if entityType == nil {
				return fmt.Errorf("entity type %q not found for %s", entitySet.EntityType, op.EntitySet)
			}

			toolName := makeToolName(system, entity, op, usedNames)
			r.mu.Lock()
			server.AddTool(r.buildTool(toolName, system, entity, op, entitySet, entityType), r.buildHandler(system, service, op, entityType))
			r.registered = append(r.registered, toolName)
			r.toolNames[operationKey(entity.ID, op.ID)] = toolName
			r.mu.Unlock()
		}
	}

	return nil
}

func (r *HierarchicalRuntime) SetActiveAccessMode(systemID, accessMode string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeSystem != systemID {
		return false
	}
	r.activeAccess = normalizeAccessMode(accessMode)
	return true
}

func (r *HierarchicalRuntime) Discover(ctx context.Context, system SystemInfo, serviceID string) (*models.DashboardServiceDiscovery, error) {
	service, ok := findService(system.Services, serviceID)
	if !ok {
		return nil, fmt.Errorf("service %q not found", serviceID)
	}

	meta, err := r.metadataFor(ctx, system, service)
	if err != nil {
		return nil, err
	}

	entitySets := make([]models.EntitySetSummary, 0, len(meta.EntitySets))
	names := make([]string, 0, len(meta.EntitySets))
	for name := range meta.EntitySets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		item := meta.EntitySets[name]
		entitySets = append(entitySets, models.EntitySetSummary{
			Name:       item.Name,
			EntityType: item.EntityType,
			Creatable:  item.Creatable || item.SAPCreatable,
			Updatable:  item.Updatable || item.SAPUpdatable,
			Deletable:  item.Deletable || item.SAPDeletable,
			Searchable: item.Searchable || item.SAPSearchable,
			Pageable:   item.Pageable || item.SAPPageable,
		})
	}

	functions := make([]models.FunctionImportBrief, 0, len(meta.FunctionImports))
	fnNames := make([]string, 0, len(meta.FunctionImports))
	for name := range meta.FunctionImports {
		fnNames = append(fnNames, name)
	}
	sort.Strings(fnNames)
	for _, name := range fnNames {
		item := meta.FunctionImports[name]
		functions = append(functions, models.FunctionImportBrief{
			Name:           item.Name,
			HTTPMethod:     item.HTTPMethod,
			ReturnType:     item.ReturnType,
			ParameterCount: len(item.Parameters),
			IsAction:       item.IsAction,
		})
	}

	return &models.DashboardServiceDiscovery{
		SystemID:        system.ID,
		ServiceID:       service.ID,
		ServiceName:     service.Name,
		ServiceURL:      service.ServiceURL,
		SafeServiceURL:  service.safeServiceURL(),
		Version:         meta.Version,
		SchemaNamespace: meta.SchemaNamespace,
		ContainerName:   meta.ContainerName,
		EntitySets:      entitySets,
		FunctionImports: functions,
	}, nil
}

func (r *HierarchicalRuntime) ToolName(entityID, operationID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.toolNames[operationKey(entityID, operationID)]
}

func (r *HierarchicalRuntime) metadataFor(ctx context.Context, system SystemInfo, service ServiceInfo) (*models.ODataMetadata, error) {
	key := cacheKey(system, service)
	r.mu.Lock()
	if meta, ok := r.metadataCache[key]; ok {
		r.serviceCacheKeys[serviceBindingKey(system.ID, service.ID)] = key
		r.mu.Unlock()
		return meta, nil
	}
	r.mu.Unlock()

	client := r.clientFor(system, service)
	meta, err := client.GetMetadata(ctx)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cached, ok := r.metadataCache[key]; ok {
		r.serviceCacheKeys[serviceBindingKey(system.ID, service.ID)] = key
		return cached, nil
	}
	r.metadataCache[key] = meta
	r.serviceCacheKeys[serviceBindingKey(system.ID, service.ID)] = key
	return meta, nil
}

func (r *HierarchicalRuntime) clientFor(system SystemInfo, service ServiceInfo) *client.ODataClient {
	key := cacheKey(system, service)
	r.mu.Lock()
	if cached, ok := r.clientCache[key]; ok {
		r.mu.Unlock()
		return cached
	}
	r.mu.Unlock()

	serviceURL := normalizeServiceURL(service.ServiceURL, system.Client)
	c := client.NewODataClient(serviceURL, r.baseConfig.Verbose)
	if system.Username != "" || system.Password != "" {
		c.SetBasicAuth(system.Username, system.Password)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cached, ok := r.clientCache[key]; ok {
		return cached
	}
	r.clientCache[key] = c
	return c
}

func (r *HierarchicalRuntime) buildTool(toolName string, system SystemInfo, entity EntityInfo, op OperationInfo, entitySet *models.EntitySet, entityType *models.EntityType) *mcp.Tool {
	verb := normalizeVerb(op.Verb)
	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           map[string]interface{}{},
		"required":             []string{},
		"additionalProperties": true,
	}
	props := schema["properties"].(map[string]interface{})
	required := make([]string, 0)

	switch verb {
	case "list":
		props["$filter"] = map[string]interface{}{"type": "string", "description": "OData $filter expression"}
		props["$select"] = map[string]interface{}{"type": "string", "description": "Comma-separated field projection"}
		props["$expand"] = map[string]interface{}{"type": "string", "description": "OData $expand expression"}
		props["$orderby"] = map[string]interface{}{"type": "string", "description": "OData $orderby expression"}
		props["$top"] = map[string]interface{}{"type": "integer", "description": "Maximum number of rows"}
		props["$skip"] = map[string]interface{}{"type": "integer", "description": "Rows to skip"}
		props["$count"] = map[string]interface{}{"type": "boolean", "description": "Return total count"}
	case "get", "delete":
		keyProps, keyRequired := keySchema(entityType)
		for name, value := range keyProps {
			props[name] = value
		}
		required = append(required, keyRequired...)
		if verb == "get" {
			props["$select"] = map[string]interface{}{"type": "string", "description": "Comma-separated field projection"}
			props["$expand"] = map[string]interface{}{"type": "string", "description": "OData $expand expression"}
		}
	case "create":
		for _, prop := range entityType.Properties {
			props[prop.Name] = propertySchema(prop)
		}
	case "update":
		keyProps, keyRequired := keySchema(entityType)
		for name, value := range keyProps {
			props[name] = value
		}
		required = append(required, keyRequired...)
		for _, prop := range entityType.Properties {
			if !contains(entityType.KeyProperties, prop.Name) {
				props[prop.Name] = propertySchema(prop)
			}
		}
		props["_method"] = map[string]interface{}{
			"type":        "string",
			"description": "Override update HTTP method (PATCH, PUT, MERGE)",
		}
	}

	schema["required"] = required
	return &mcp.Tool{
		Name:        toolName,
		Description: toolDescription(system, entity, op, entitySet),
		InputSchema: schema,
	}
}

func (r *HierarchicalRuntime) buildHandler(system SystemInfo, service ServiceInfo, op OperationInfo, entityType *models.EntityType) mcp.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		verb := normalizeVerb(op.Verb)
		if isMutatingVerb(verb) && r.currentAccessMode(system.ID) == "restricted" {
			return nil, fmt.Errorf("write operations are not allowed while system %q is in restricted mode", system.Name)
		}
		client := r.clientFor(system, service)
		switch verb {
		case "list":
			resp, err := client.GetEntitySet(ctx, op.EntitySet, operationQueryOptions(op, args))
			return marshalResponse(resp, err)
		case "get":
			resp, err := client.GetEntity(ctx, op.EntitySet, buildKeyMap(entityType, args), operationQueryOptions(op, args))
			return marshalResponse(resp, err)
		case "create":
			payload := stripSystemArgs(args)
			payload = utils.ConvertNumericsInMap(payload)
			if r.baseConfig.LegacyDates {
				payload = utils.ConvertDatesInMap(payload, false)
			}
			resp, err := client.CreateEntity(ctx, op.EntitySet, payload)
			return marshalResponse(resp, err)
		case "update":
			key := buildKeyMap(entityType, args)
			payload := stripSystemArgs(args)
			for _, keyName := range entityType.KeyProperties {
				delete(payload, keyName)
			}
			method := constants.PUT
			if raw, ok := args["_method"].(string); ok && strings.TrimSpace(raw) != "" {
				method = strings.ToUpper(strings.TrimSpace(raw))
			}
			payload = utils.ConvertNumericsInMap(payload)
			if r.baseConfig.LegacyDates {
				payload = utils.ConvertDatesInMap(payload, false)
			}
			resp, err := client.UpdateEntity(ctx, op.EntitySet, key, payload, method)
			return marshalResponse(resp, err)
		case "delete":
			_, err := client.DeleteEntity(ctx, op.EntitySet, buildKeyMap(entityType, args))
			if err != nil {
				return nil, err
			}
			return `{"status":"success","message":"Entity deleted successfully"}`, nil
		default:
			return nil, fmt.Errorf("unsupported operation verb %q", op.Verb)
		}
	}
}

func (r *HierarchicalRuntime) currentAccessMode(systemID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeSystem == systemID && r.activeAccess != "" {
		return r.activeAccess
	}
	return "unrestricted"
}

func cacheKey(system SystemInfo, service ServiceInfo) string {
	return system.ID + "::" + normalizeServiceURL(service.ServiceURL, system.Client) + "::" + system.Username
}

func queryOptions(args map[string]interface{}) map[string]string {
	options := make(map[string]string)
	for key, value := range args {
		switch v := value.(type) {
		case string:
			if strings.HasPrefix(key, "$") && strings.TrimSpace(v) != "" {
				options[key] = v
			}
		case float64:
			if strings.HasPrefix(key, "$") {
				options[key] = fmt.Sprintf("%d", int(v))
			}
		case bool:
			if key == "$count" && v {
				options[key] = "true"
			}
		}
	}
	return options
}

func operationQueryOptions(op OperationInfo, args map[string]interface{}) map[string]string {
	options := copyQueryOptions(op.Query)
	if options == nil {
		options = make(map[string]string)
	}
	for key, value := range queryOptions(args) {
		options[key] = value
	}
	return options
}

func stripSystemArgs(args map[string]interface{}) map[string]interface{} {
	payload := make(map[string]interface{}, len(args))
	for key, value := range args {
		if strings.HasPrefix(key, "$") || key == "_method" {
			continue
		}
		payload[key] = value
	}
	return payload
}

func buildKeyMap(entityType *models.EntityType, args map[string]interface{}) map[string]interface{} {
	key := make(map[string]interface{}, len(entityType.KeyProperties))
	for _, keyProp := range entityType.KeyProperties {
		value, ok := args[keyProp]
		if !ok {
			continue
		}
		if prop := findProperty(entityType, keyProp); prop != nil && prop.Type == "Edm.Guid" {
			if s, ok := value.(string); ok {
				key[keyProp] = models.GUIDValue(s)
				continue
			}
		}
		key[keyProp] = value
	}
	return key
}

func keySchema(entityType *models.EntityType) (map[string]interface{}, []string) {
	props := make(map[string]interface{})
	required := make([]string, 0, len(entityType.KeyProperties))
	for _, keyProp := range entityType.KeyProperties {
		prop := findProperty(entityType, keyProp)
		if prop == nil {
			continue
		}
		props[keyProp] = propertySchema(prop)
		required = append(required, keyProp)
	}
	return props, required
}

func propertySchema(prop *models.EntityProperty) map[string]interface{} {
	return map[string]interface{}{
		"type":        jsonSchemaType(prop.Type),
		"description": prop.Name,
	}
}

func jsonSchemaType(odataType string) string {
	switch odataType {
	case "Edm.String", "Edm.Guid", "Edm.DateTime", "Edm.DateTimeOffset", "Edm.Time", "Edm.Binary":
		return "string"
	case "Edm.Int16", "Edm.Int32", "Edm.Int64", "Edm.Byte", "Edm.SByte":
		return "integer"
	case "Edm.Single", "Edm.Double", "Edm.Decimal":
		return "number"
	case "Edm.Boolean":
		return "boolean"
	default:
		return "string"
	}
}

func lookupEntityType(meta *models.ODataMetadata, typeName string) *models.EntityType {
	if entityType, ok := meta.EntityTypes[typeName]; ok {
		return entityType
	}
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		return meta.EntityTypes[typeName[idx+1:]]
	}
	return nil
}

func findProperty(entityType *models.EntityType, name string) *models.EntityProperty {
	for _, prop := range entityType.Properties {
		if prop.Name == name {
			return prop
		}
	}
	return nil
}

func makeToolName(system SystemInfo, entity EntityInfo, op OperationInfo, used map[string]struct{}) string {
	entityPart := toolNamePart(entity.ID)
	if entityPart == "" {
		entityPart = toolNamePart(entity.Label)
	}
	if entityPart == "" {
		entityPart = "entity"
	}

	opPart := toolNamePart(op.ID)
	if opPart == "" {
		opPart = toolNamePart(op.Name)
	}
	if opPart == "" {
		opPart = toolNamePart(normalizeVerb(op.Verb) + "_" + op.EntitySet)
	}
	prefix := entityPart + "_"
	opPart = strings.TrimPrefix(opPart, prefix)
	if opPart == "" {
		opPart = normalizeVerb(op.Verb)
	}

	base := entityPart + "_" + opPart + "_for_" + toolNamePart(system.ID)
	if _, exists := used[base]; !exists {
		used[base] = struct{}{}
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%d", base, i)
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func toolNamePart(raw string) string {
	return strings.ReplaceAll(slugify(raw), "-", "_")
}

func toolDescription(system SystemInfo, entity EntityInfo, op OperationInfo, entitySet *models.EntitySet) string {
	name := strings.TrimSpace(op.Name)
	if name == "" {
		name = defaultOperationName(op)
	}
	return fmt.Sprintf("%s for %s in system %s via %s", name, entity.Label, system.Name, entitySet.Name)
}

func marshalResponse(resp *models.ODataResponse, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return string(body), nil
}

func isMutatingVerb(verb string) bool {
	switch normalizeVerb(verb) {
	case "create", "update", "delete":
		return true
	default:
		return false
	}
}

func findService(services []ServiceInfo, id string) (ServiceInfo, bool) {
	for _, service := range services {
		if service.ID == id {
			return service, true
		}
	}
	return ServiceInfo{}, false
}

func operationKey(entityID, operationID string) string {
	return entityID + "::" + operationID
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
