package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/mcp"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
)

const cachedSummaryToolName = "odata_cached_csdl_summary"

func (r *HierarchicalRuntime) registerCachedCSDLSummaryTool() {
	r.bridge.GetServer().AddTool(&mcp.Tool{
		Name:        cachedSummaryToolName,
		Description: "Return a sanitized cached CSDL contract: SHA-256 digest, version, EntitySet to EntityType mappings, keys, property types/facets, and deterministic function import summaries. Reads runtime memory only and never performs an OData or network request.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"system_id":  map[string]interface{}{"type": "string"},
				"service_id": map[string]interface{}{"type": "string"},
			},
			"required":             []string{"system_id", "service_id"},
			"additionalProperties": false,
		},
		OutputSchema: cachedCSDLSummaryOutputSchema(),
	}, r.cachedCSDLSummaryHandler)
}

func (r *HierarchicalRuntime) cachedCSDLSummaryHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	systemID, _ := args["system_id"].(string)
	serviceID, _ := args["service_id"].(string)
	systemID = strings.TrimSpace(systemID)
	serviceID = strings.TrimSpace(serviceID)
	if systemID == "" || serviceID == "" {
		return nil, fmt.Errorf("system_id and service_id are required")
	}

	metadata, err := r.cachedMetadata(ctx, systemID, serviceID)
	if err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return buildCachedCSDLSummary(ctx, metadata)
}

func (r *HierarchicalRuntime) cachedMetadata(ctx context.Context, systemID, serviceID string) (*models.ODataMetadata, error) {
	if err := lockRuntimeWithContext(ctx, &r.mu); err != nil {
		return nil, err
	}
	defer r.mu.Unlock()
	key, ok := r.serviceCacheKeys[serviceBindingKey(systemID, serviceID)]
	if !ok {
		return nil, fmt.Errorf("cached metadata unavailable; activation is required")
	}
	metadata, ok := r.metadataCache[key]
	if !ok || metadata == nil {
		return nil, fmt.Errorf("cached metadata unavailable; activation is required")
	}
	return metadata, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func lockRuntimeWithContext(ctx context.Context, mu *sync.Mutex) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if mu.TryLock() {
			return nil
		}
		timer := time.NewTimer(time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func serviceBindingKey(systemID, serviceID string) string {
	return systemID + "::" + serviceID
}

func cachedCSDLSummaryOutputSchema() map[string]interface{} {
	propertySchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":       map[string]interface{}{"type": "string"},
			"type":       map[string]interface{}{"type": "string"},
			"nullable":   map[string]interface{}{"type": "boolean"},
			"is_key":     map[string]interface{}{"type": "boolean"},
			"max_length": map[string]interface{}{"type": "string"},
			"precision":  map[string]interface{}{"type": "string"},
			"scale":      map[string]interface{}{"type": "string"},
		},
		"required":             []string{"name", "type", "nullable", "is_key"},
		"additionalProperties": false,
	}
	functionImportSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":            map[string]interface{}{"type": "string"},
			"http_method":     map[string]interface{}{"type": "string"},
			"return_type":     map[string]interface{}{"type": "string"},
			"parameter_count": map[string]interface{}{"type": "integer", "minimum": 0},
			"is_action":       map[string]interface{}{"type": "boolean"},
		},
		"required":             []string{"name", "http_method", "return_type", "parameter_count", "is_action"},
		"additionalProperties": false,
	}
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"digest_sha256":    map[string]interface{}{"type": "string"},
			"version":          map[string]interface{}{"type": "string"},
			"schema_namespace": map[string]interface{}{"type": "string"},
			"container_name":   map[string]interface{}{"type": "string"},
			"entity_sets": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":        map[string]interface{}{"type": "string"},
						"entity_type": map[string]interface{}{"type": "string"},
						"keys":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
						"properties":  map[string]interface{}{"type": "array", "items": propertySchema},
					},
					"required":             []string{"name", "entity_type", "keys", "properties"},
					"additionalProperties": false,
				},
			},
			"function_imports": map[string]interface{}{
				"type":  "array",
				"items": functionImportSchema,
			},
		},
		"required":             []string{"digest_sha256", "version", "schema_namespace", "container_name", "entity_sets", "function_imports"},
		"additionalProperties": false,
	}
}

type cachedCSDLProperty struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Nullable  bool   `json:"nullable"`
	IsKey     bool   `json:"is_key"`
	MaxLength string `json:"max_length,omitempty"`
	Precision string `json:"precision,omitempty"`
	Scale     string `json:"scale,omitempty"`
}

type cachedCSDLEntitySet struct {
	Name       string               `json:"name"`
	EntityType string               `json:"entity_type"`
	Keys       []string             `json:"keys"`
	Properties []cachedCSDLProperty `json:"properties"`
}

type cachedCSDLFunctionImport struct {
	Name           string `json:"name"`
	HTTPMethod     string `json:"http_method"`
	ReturnType     string `json:"return_type"`
	ParameterCount int    `json:"parameter_count"`
	IsAction       bool   `json:"is_action"`
}

type cachedCSDLPayload struct {
	Version         string                     `json:"version"`
	SchemaNamespace string                     `json:"schema_namespace"`
	ContainerName   string                     `json:"container_name"`
	EntitySets      []cachedCSDLEntitySet      `json:"entity_sets"`
	FunctionImports []cachedCSDLFunctionImport `json:"function_imports"`
}

type cachedCSDLSummary struct {
	DigestSHA256 string `json:"digest_sha256"`
	cachedCSDLPayload
}

func buildCachedCSDLSummary(ctx context.Context, metadata *models.ODataMetadata) (*cachedCSDLSummary, error) {
	if metadata == nil {
		return nil, fmt.Errorf("cached metadata unavailable")
	}

	payload := cachedCSDLPayload{
		Version:         metadata.Version,
		SchemaNamespace: metadata.SchemaNamespace,
		ContainerName:   metadata.ContainerName,
		EntitySets:      make([]cachedCSDLEntitySet, 0, len(metadata.EntitySets)),
		FunctionImports: make([]cachedCSDLFunctionImport, 0, len(metadata.FunctionImports)),
	}
	entitySetNames := make([]string, 0, len(metadata.EntitySets))
	for name := range metadata.EntitySets {
		entitySetNames = append(entitySetNames, name)
	}
	sort.Strings(entitySetNames)

	for _, name := range entitySetNames {
		if err := contextError(ctx); err != nil {
			return nil, err
		}
		entitySet := metadata.EntitySets[name]
		if entitySet == nil {
			return nil, fmt.Errorf("cached metadata has an empty entity set %q", name)
		}
		entityType := lookupEntityType(metadata, entitySet.EntityType)
		if entityType == nil {
			return nil, fmt.Errorf("cached metadata has no entity type %q for entity set %q", entitySet.EntityType, name)
		}

		keys := append([]string(nil), entityType.KeyProperties...)
		sort.Strings(keys)
		properties := make([]cachedCSDLProperty, 0, len(entityType.Properties))
		for _, property := range entityType.Properties {
			if err := contextError(ctx); err != nil {
				return nil, err
			}
			if property == nil {
				continue
			}
			properties = append(properties, cachedCSDLProperty{
				Name:      property.Name,
				Type:      property.Type,
				Nullable:  property.Nullable,
				IsKey:     property.IsKey,
				MaxLength: property.MaxLength,
				Precision: property.Precision,
				Scale:     property.Scale,
			})
		}
		sort.Slice(properties, func(i, j int) bool {
			return properties[i].Name < properties[j].Name
		})
		payload.EntitySets = append(payload.EntitySets, cachedCSDLEntitySet{
			Name:       entitySet.Name,
			EntityType: entitySet.EntityType,
			Keys:       keys,
			Properties: properties,
		})
	}

	functionImportNames := make([]string, 0, len(metadata.FunctionImports))
	for name := range metadata.FunctionImports {
		functionImportNames = append(functionImportNames, name)
	}
	sort.Strings(functionImportNames)
	for _, name := range functionImportNames {
		if err := contextError(ctx); err != nil {
			return nil, err
		}
		functionImport := metadata.FunctionImports[name]
		if functionImport == nil {
			return nil, fmt.Errorf("cached metadata has an empty function import %q", name)
		}
		payload.FunctionImports = append(payload.FunctionImports, cachedCSDLFunctionImport{
			Name:           functionImport.Name,
			HTTPMethod:     functionImport.HTTPMethod,
			ReturnType:     functionImport.ReturnType,
			ParameterCount: len(functionImport.Parameters),
			IsAction:       functionImport.IsAction,
		})
	}
	sort.SliceStable(payload.FunctionImports, func(i, j int) bool {
		left, right := payload.FunctionImports[i], payload.FunctionImports[j]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		if left.HTTPMethod != right.HTTPMethod {
			return left.HTTPMethod < right.HTTPMethod
		}
		if left.ReturnType != right.ReturnType {
			return left.ReturnType < right.ReturnType
		}
		if left.ParameterCount != right.ParameterCount {
			return left.ParameterCount < right.ParameterCount
		}
		return !left.IsAction && right.IsAction
	})

	if err := contextError(ctx); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode sanitized CSDL summary: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return &cachedCSDLSummary{
		DigestSHA256:      hex.EncodeToString(digest[:]),
		cachedCSDLPayload: payload,
	}, nil
}
