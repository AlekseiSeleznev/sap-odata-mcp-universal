package dashboard

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type ServiceInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ServiceURL string `json:"service_url"`
}

func (s ServiceInfo) safeServiceURL() string {
	parsed, err := url.Parse(s.ServiceURL)
	if err != nil {
		return s.ServiceURL
	}
	parsed.User = nil
	return parsed.String()
}

type OperationInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name,omitempty"`
	Verb      string            `json:"verb"`
	ServiceID string            `json:"service_id"`
	EntitySet string            `json:"entity_set"`
	Query     map[string]string `json:"query,omitempty"`
	Mode      string            `json:"mode"`
	Enabled   bool              `json:"enabled"`
}

type EntityInfo struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	Description string          `json:"description,omitempty"`
	Operations  []OperationInfo `json:"operations"`
}

type SystemInfo struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	BaseURL    string        `json:"base_url,omitempty"`
	Client     string        `json:"client,omitempty"`
	Username   string        `json:"username"`
	Password   string        `json:"password,omitempty"`
	AccessMode string        `json:"access_mode"`
	Connected  bool          `json:"connected,omitempty"`
	Services   []ServiceInfo `json:"services"`
	Entities   []EntityInfo  `json:"entities"`
}

func (s SystemInfo) toPersisted() SystemInfo {
	copy := s
	copy.Connected = false
	return copy
}

type legacyConnectionInfo struct {
	Name       string `json:"name"`
	SystemName string `json:"system_name"`
	ServiceURL string `json:"service_url"`
	Client     string `json:"client,omitempty"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	AccessMode string `json:"access_mode"`
}

type legacyRegistryState struct {
	Active      string                 `json:"active"`
	Connections []legacyConnectionInfo `json:"connections"`
}

type registryState struct {
	SchemaVersion int          `json:"schema_version"`
	ActiveSystem  string       `json:"active_system"`
	Systems       []SystemInfo `json:"systems"`
}

type Registry struct {
	path         string
	activeSystem string
	systems      map[string]SystemInfo
	mu           sync.Mutex
}

func NewConnectionRegistry(path string) *Registry {
	return &Registry{
		path:    path,
		systems: make(map[string]SystemInfo),
	}
}

func DefaultStateFile() string {
	if env := strings.TrimSpace(os.Getenv("ODATA_MCP_STATE_FILE")); env != "" {
		return env
	}
	if _, err := os.Stat("/data"); err == nil {
		return "/data/odata_state.json"
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "sap-odata-mcp-universal-state.json")
	}
	return filepath.Join(configDir, "sap-odata-mcp-universal", "odata_state.json")
}

func (r *Registry) Load() []SystemInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil
	}

	var state registryState
	if err := json.Unmarshal(data, &state); err == nil && state.SchemaVersion >= 2 {
		r.activeSystem = state.ActiveSystem
		r.systems = make(map[string]SystemInfo, len(state.Systems))
		for _, system := range state.Systems {
			system.Connected = false
			r.systems[system.ID] = sanitizeSystem(system)
		}
		return r.listLocked()
	}

	var legacy legacyRegistryState
	if err := json.Unmarshal(data, &legacy); err != nil {
		log.Printf("dashboard registry: failed to parse state file %s: %v", r.path, err)
		return nil
	}

	migrated, active := migrateLegacyConnections(legacy)
	r.activeSystem = active
	r.systems = make(map[string]SystemInfo, len(migrated))
	for _, system := range migrated {
		r.systems[system.ID] = system
	}
	r.saveLocked()
	return r.listLocked()
}

func (r *Registry) Active() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.activeSystem
}

func (r *Registry) Path() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.path
}

func (r *Registry) Get(id string) (SystemInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	system, ok := r.systems[id]
	return system, ok
}

func (r *Registry) ListAll() []SystemInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listLocked()
}

func (r *Registry) ReplaceLoadedSystems(systems []SystemInfo, active string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.systems = make(map[string]SystemInfo, len(systems))
	for _, system := range systems {
		r.systems[system.ID] = sanitizeSystem(system)
	}
	r.activeSystem = active
	r.saveLocked()
}

func (r *Registry) listLocked() []SystemInfo {
	result := make([]SystemInfo, 0, len(r.systems))
	for _, system := range r.systems {
		result = append(result, sanitizeSystem(system))
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result
}

func (r *Registry) saveLocked() {
	state := registryState{
		SchemaVersion: 2,
		ActiveSystem:  r.activeSystem,
		Systems:       r.listLocked(),
	}
	for i := range state.Systems {
		state.Systems[i] = state.Systems[i].toPersisted()
	}

	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		log.Printf("dashboard registry: failed to create state dir for %s: %v", r.path, err)
		return
	}

	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Printf("dashboard registry: failed to marshal state: %v", err)
		return
	}

	if err := os.WriteFile(r.path, body, 0o600); err != nil {
		log.Printf("dashboard registry: failed to write state file %s: %v", r.path, err)
	}
}

func sanitizeSystem(system SystemInfo) SystemInfo {
	system.AccessMode = normalizeAccessMode(system.AccessMode)
	system.Services = sortedServices(system.Services)
	system.Entities = sortedEntities(system.Entities)
	return system
}

func sortedServices(items []ServiceInfo) []ServiceInfo {
	copyItems := append([]ServiceInfo(nil), items...)
	sort.Slice(copyItems, func(i, j int) bool {
		return strings.ToLower(copyItems[i].Name) < strings.ToLower(copyItems[j].Name)
	})
	return copyItems
}

func sortedEntities(items []EntityInfo) []EntityInfo {
	copyItems := append([]EntityInfo(nil), items...)
	sort.Slice(copyItems, func(i, j int) bool {
		return strings.ToLower(copyItems[i].Label) < strings.ToLower(copyItems[j].Label)
	})
	for i := range copyItems {
		sort.Slice(copyItems[i].Operations, func(a, b int) bool {
			left := copyItems[i].Operations[a]
			right := copyItems[i].Operations[b]
			leftRank := operationRank(left.Verb)
			rightRank := operationRank(right.Verb)
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			return strings.ToLower(operationSortLabel(left)) < strings.ToLower(operationSortLabel(right))
		})
	}
	return copyItems
}

func operationSortLabel(op OperationInfo) string {
	if strings.TrimSpace(op.Name) != "" {
		return op.Name
	}
	if strings.TrimSpace(op.EntitySet) != "" {
		return op.EntitySet
	}
	return op.ID
}

func operationRank(verb string) int {
	switch normalizeVerb(verb) {
	case "get":
		return 0
	case "list":
		return 1
	case "create":
		return 2
	case "update":
		return 3
	case "delete":
		return 4
	default:
		return 10
	}
}

func normalizeAccessMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "restricted") {
		return "restricted"
	}
	return "unrestricted"
}

func normalizeVerb(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "get":
		return "get"
	case "list":
		return "list"
	case "post", "create":
		return "create"
	case "put", "patch", "merge", "update":
		return "update"
	case "delete", "remove":
		return "delete"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "composite") {
		return "composite"
	}
	return "generated"
}

func slugify(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(raw) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		b.WriteByte('-')
		lastDash = true
	}

	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "item"
	}
	return result
}

func ensureUniqueID(candidate string, exists func(string) bool) string {
	candidate = slugify(candidate)
	if candidate == "" {
		candidate = "item"
	}
	if !exists(candidate) {
		return candidate
	}
	for i := 2; ; i++ {
		tryID := candidate + "-" + strconv.Itoa(i)
		if !exists(tryID) {
			return tryID
		}
	}
}

func deriveBaseURL(serviceURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(serviceURL))
	if err != nil {
		return ""
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func migrateLegacyConnections(legacy legacyRegistryState) ([]SystemInfo, string) {
	type groupKey struct {
		name     string
		client   string
		username string
	}

	grouped := make(map[groupKey][]legacyConnectionInfo)
	order := make([]groupKey, 0)
	for _, conn := range legacy.Connections {
		key := groupKey{
			name:     strings.TrimSpace(conn.SystemName),
			client:   strings.TrimSpace(conn.Client),
			username: strings.TrimSpace(conn.Username),
		}
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], conn)
	}

	systems := make([]SystemInfo, 0, len(order))
	active := ""
	legacyToSystem := make(map[string]string)

	for _, key := range order {
		conns := grouped[key]
		first := conns[0]
		systemName := strings.TrimSpace(first.SystemName)
		if systemName == "" {
			systemName = strings.TrimSpace(first.Name)
		}

		systemID := ensureUniqueID(systemName+"-"+first.Client, func(id string) bool {
			for _, system := range systems {
				if system.ID == id {
					return true
				}
			}
			return false
		})

		system := SystemInfo{
			ID:         systemID,
			Name:       systemName,
			BaseURL:    deriveBaseURL(first.ServiceURL),
			Client:     strings.TrimSpace(first.Client),
			Username:   strings.TrimSpace(first.Username),
			Password:   first.Password,
			AccessMode: normalizeAccessMode(first.AccessMode),
			Services:   make([]ServiceInfo, 0, len(conns)),
			Entities:   []EntityInfo{},
		}

		for _, conn := range conns {
			serviceID := ensureUniqueID(conn.Name, func(id string) bool {
				for _, service := range system.Services {
					if service.ID == id {
						return true
					}
				}
				return false
			})
			system.Services = append(system.Services, ServiceInfo{
				ID:         serviceID,
				Name:       strings.TrimSpace(conn.Name),
				ServiceURL: strings.TrimSpace(conn.ServiceURL),
			})
			legacyToSystem[conn.Name] = systemID
		}

		systems = append(systems, sanitizeSystem(system))
	}

	if legacy.Active != "" {
		active = legacyToSystem[legacy.Active]
	}
	if active == "" && len(systems) > 0 {
		active = systems[0].ID
	}

	return systems, active
}
