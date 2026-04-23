package dashboard

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type ConnectionInfo struct {
	Name       string `json:"name"`
	SystemName string `json:"system_name"`
	ServiceURL string `json:"service_url"`
	Client     string `json:"client,omitempty"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	AccessMode string `json:"access_mode"`
	Connected  bool   `json:"connected"`
}

func (c ConnectionInfo) safeServiceURL() string {
	parsed, err := url.Parse(c.ServiceURL)
	if err != nil {
		return c.ServiceURL
	}
	parsed.User = nil
	return parsed.String()
}

func (c ConnectionInfo) detailLine() string {
	parts := []string{c.safeServiceURL()}
	if c.Client != "" {
		parts = append(parts, "client="+c.Client)
	}
	if c.Username != "" {
		parts = append(parts, c.Username)
	}
	return strings.Join(parts, " | ")
}

func (c ConnectionInfo) toPersisted() ConnectionInfo {
	copy := c
	copy.Connected = false
	return copy
}

type registryState struct {
	Active      string           `json:"active"`
	Connections []ConnectionInfo `json:"connections"`
}

type ConnectionRegistry struct {
	path        string
	active      string
	connections map[string]ConnectionInfo
	mu          sync.Mutex
}

func NewConnectionRegistry(path string) *ConnectionRegistry {
	return &ConnectionRegistry{
		path:        path,
		connections: make(map[string]ConnectionInfo),
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
		return filepath.Join(os.TempDir(), "odata-mcp-universal-state.json")
	}
	return filepath.Join(configDir, "odata-mcp-universal", "odata_state.json")
}

func (r *ConnectionRegistry) Load() []ConnectionInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil
	}

	var state registryState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("dashboard registry: failed to parse state file %s: %v", r.path, err)
		return nil
	}

	r.active = state.Active
	r.connections = make(map[string]ConnectionInfo, len(state.Connections))
	for _, conn := range state.Connections {
		conn.Connected = false
		r.connections[conn.Name] = conn
	}

	return r.listLocked()
}

func (r *ConnectionRegistry) Save() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveLocked()
}

func (r *ConnectionRegistry) Active() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

func (r *ConnectionRegistry) Path() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.path
}

func (r *ConnectionRegistry) SetActive(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = name
	r.saveLocked()
}

func (r *ConnectionRegistry) Add(conn ConnectionInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections[conn.Name] = conn
	if r.active == "" {
		r.active = conn.Name
	}
	r.saveLocked()
}

func (r *ConnectionRegistry) Remove(name string) (ConnectionInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, ok := r.connections[name]
	if !ok {
		return ConnectionInfo{}, false
	}
	delete(r.connections, name)
	if r.active == name {
		r.active = ""
		for key := range r.connections {
			r.active = key
			break
		}
	}
	r.saveLocked()
	return conn, true
}

func (r *ConnectionRegistry) Get(name string) (ConnectionInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	conn, ok := r.connections[name]
	return conn, ok
}

func (r *ConnectionRegistry) ListAll() []ConnectionInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listLocked()
}

func (r *ConnectionRegistry) Update(name string, conn ConnectionInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.connections, name)
	r.connections[conn.Name] = conn
	if r.active == "" || r.active == name {
		r.active = conn.Name
	}
	r.saveLocked()
}

func (r *ConnectionRegistry) SetConnected(name string, connected bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	conn, ok := r.connections[name]
	if !ok {
		return
	}
	conn.Connected = connected
	r.connections[name] = conn
	r.saveLocked()
}

func (r *ConnectionRegistry) ReplaceLoadedConnections(conns []ConnectionInfo, active string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections = make(map[string]ConnectionInfo, len(conns))
	for _, conn := range conns {
		r.connections[conn.Name] = conn
	}
	r.active = active
	r.saveLocked()
}

func (r *ConnectionRegistry) listLocked() []ConnectionInfo {
	result := make([]ConnectionInfo, 0, len(r.connections))
	for _, conn := range r.connections {
		result = append(result, conn)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result
}

func (r *ConnectionRegistry) saveLocked() {
	state := registryState{
		Active:      r.active,
		Connections: r.listLocked(),
	}
	for i := range state.Connections {
		state.Connections[i] = state.Connections[i].toPersisted()
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
