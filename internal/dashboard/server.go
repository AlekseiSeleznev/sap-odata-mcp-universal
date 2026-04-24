package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
)

type Server struct {
	provider Provider
}

func New(provider Provider) *Server {
	return &Server{provider: provider}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/dashboard", s.handleIndex)
	mux.HandleFunc("/dashboard/", s.handleDashboardRoot)
	mux.HandleFunc("/dashboard/docs", s.handleDocs)

	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/systems", s.handleSystems)
	mux.HandleFunc("/api/system/test", s.handleTestSystem)
	mux.HandleFunc("/api/system/save", s.handleSaveSystem)
	mux.HandleFunc("/api/system/delete", s.handleDeleteSystem)
	mux.HandleFunc("/api/system/activate", s.handleActivateSystem)
	mux.HandleFunc("/api/service/save", s.handleSaveService)
	mux.HandleFunc("/api/service/delete", s.handleDeleteService)
	mux.HandleFunc("/api/service/discover", s.handleDiscoverService)
	mux.HandleFunc("/api/entity/save", s.handleSaveEntity)
	mux.HandleFunc("/api/entity/delete", s.handleDeleteEntity)
	mux.HandleFunc("/api/operation/save", s.handleSaveOperation)
	mux.HandleFunc("/api/operation/delete", s.handleDeleteOperation)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}

func (s *Server) handleDashboardRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/dashboard/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.URL.Path != "/dashboard" {
		http.NotFound(w, r)
		return
	}
	body, err := renderDashboard(r.URL.Query().Get("lang"))
	if err != nil {
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, err := s.provider.DocsContext(r.Context())
	if err != nil {
		http.Error(w, "failed to load docs context", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderDocs(r.URL.Query().Get("lang"), ctx)))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status, err := s.provider.Status(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSystems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items, err := s.provider.Systems(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleTestSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.TestSystem(r.Context(), req.SystemID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSaveSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardSystemUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.SaveSystem(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleDeleteSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.DeleteSystem(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleActivateSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.ActivateSystem(r.Context(), req.SystemID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleSaveService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardServiceUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.SaveService(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.DeleteService(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleDiscoverService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	systemID := r.URL.Query().Get("system_id")
	serviceID := r.URL.Query().Get("service_id")
	if systemID == "" || serviceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "system_id and service_id are required"})
		return
	}
	result, err := s.provider.DiscoverService(r.Context(), systemID, serviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSaveEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardEntityUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.SaveEntity(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleDeleteEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.DeleteEntity(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleSaveOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardOperationUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.SaveOperation(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func (s *Server) handleDeleteOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.DashboardDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := s.provider.DeleteOperation(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeMutationJSON(w, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeMutationJSON(w http.ResponseWriter, result *models.DashboardMutationResult) {
	if result == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "empty mutation result"})
		return
	}
	if !result.OK {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
