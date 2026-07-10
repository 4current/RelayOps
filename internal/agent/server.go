package agent

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
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
