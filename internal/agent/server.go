package agent

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Server struct {
	ctx context.Context
}

func NewServer(ctx context.Context) *Server {
	return &Server{ctx: ctx}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/status", s.handleStatus)
	mux.HandleFunc("/v1/services", s.handleServices)
	mux.HandleFunc("GET /v1/services/{id}", s.handleService)
	mux.HandleFunc("PUT /v1/services/{id}", s.handlePutService)
	return mux
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	status := CollectStatus(r.Context())
	writeJSON(w, http.StatusOK, status)

}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	status := CollectStatus(r.Context())
	writeJSON(w, http.StatusOK, status.Services)

}

type UpdateServiceRequest struct {
	DesiredState string `json:"desired_state"`
}

type UpdateServiceResponse struct {
	Service      string        `json:"service"`
	DesiredState string        `json:"desired_state"`
	Status       ServiceStatus `json:"status"`
	Output       string        `json:"output,omitempty"`
}

func (s *Server) handleService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, ok := ServiceCatalog[id]; !ok {
		writeError(w, http.StatusNotFound, "unknown service")
		return
	}

	facts := CollectFacts(r.Context())
	services := EvaluateServices(facts)

	status, ok := services[id]
	if !ok {
		writeError(w, http.StatusNotFound, "service status unavailable")
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handlePutService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	def, ok := ServiceCatalog[id]
	if !ok {
		writeError(w, http.StatusNotFound, "unknown service")
		return
	}

	var req UpdateServiceRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var cmd []string

	switch req.DesiredState {
	case "running":
		cmd = def.StartCmd
	case "stopped":
		cmd = def.StopCmd
	default:
		writeError(
			w,
			http.StatusBadRequest,
			`desired_state must be "running" or "stopped"`,
		)
		return
	}

	if len(cmd) == 0 {
		writeError(w, http.StatusConflict, "service has no command for requested state")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	output, err := runServiceCommand(ctx, cmd)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, struct {
			Error   string `json:"error"`
			Service string `json:"service"`
			Output  string `json:"output,omitempty"`
		}{
			Error:   err.Error(),
			Service: id,
			Output:  output,
		})
		return
	}

	// Re-probe after the command completes.
	facts := CollectFacts(r.Context())
	statuses := EvaluateServices(facts)
	status := statuses[id]

	writeJSON(w, http.StatusOK, UpdateServiceResponse{
		Service:      id,
		DesiredState: req.DesiredState,
		Status:       status,
		Output:       output,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		// At this point the response may already have started, so log rather
		// than trying to write another HTTP response.
		log.Printf("encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, struct {
		Error string `json:"error"`
	}{
		Error: message,
	})
}
