package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"visor/internal/config"
)

type Server struct {
	cfg *config.Config
	mux *http.ServeMux
}

func New(cfg *config.Config) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /webhook", s.handleWebhook)
	return s
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("visor listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// stub — will be wired up in M1 iteration 2
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}
