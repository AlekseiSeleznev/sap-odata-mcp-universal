package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/mcp"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
)

const cachedSummaryToolName = "odata_cached_csdl_summary"

func (r *HierarchicalRuntime) registerCachedCSDLSummaryTool() {
	r.bridge.GetServer().AddTool(&mcp.Tool{
		Name:        cachedSummaryToolName,
		Description: "Return a sanitized cached CSDL contract: SHA-256 digest, version, EntitySet to EntityType mappings, keys, and property types/facets. Reads runtime memory only and never performs an OData or network request.",
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

func (r *HierarchicalRuntime) cachedCSDLSummaryHandler(_ context.Context, args map[string]interface{}) (interface{}, error) {
	systemID, _ := args["system_id"].(string)
	serviceID, _ := args["service_id"].(string)
	systemID = strings.TrimSpace(systemID)
	serviceID = strings.TrimSpace(serviceID)
	if systemID == "" || serviceID == "" {
		return nil, fmt.Errorf("system_id and service_id are required")
	}

	metadata, err := r.cachedMetadata(systemID, serviceID)
	if err != nil {
		return nil, err
	}
	return buildCachedCSDLSummary(metadata)
}

func (r *HierarchicalRuntime) cachedMetadata(systemID, serviceID string) (*models.ODataMetadata, error) {
	r.mu.Lock()
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
		},
		"required":             []string{"digest_sha256", "version", "schema_namespace", "container_name", "entity_sets"},
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

type cachedCSDLPayload struct {
	Version         string                `json:"version"`
	SchemaNamespace string                `json:"schema_namespace"`
	ContainerName   string                `json:"container_name"`
	EntitySets      []cachedCSDLEntitySet `json:"entity_sets"`
}

type cachedCSDLSummary struct {
	DigestSHA256 string `json:"digest_sha256"`
	cachedCSDLPayload
}

func buildCachedCSDLSummary(metadata *models.ODataMetadata) (*cachedCSDLSummary, error) {
	if metadata == nil {
		return nil, fmt.Errorf("cached metadata unavailable")
	}

	payload := cachedCSDLPayload{
		Version:         metadata.Version,
		SchemaNamespace: metadata.SchemaNamespace,
		ContainerName:   metadata.ContainerName,
		EntitySets:      make([]cachedCSDLEntitySet, 0, len(metadata.EntitySets)),
	}
	entitySetNames := make([]string, 0, len(metadata.EntitySets))
	for name := range metadata.EntitySets {
		entitySetNames = append(entitySetNames, name)
	}
	sort.Strings(entitySetNames)

	for _, name := range entitySetNames {
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
