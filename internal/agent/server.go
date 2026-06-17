package agent

import (
	"context"
	"encoding/json"
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
	mux.HandleFunc("/status", s.handleStatus)
	return mux
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := CollectStatus(s.ctx)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
