package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zhulik/fid/pkg/log"
)

type Server struct {
	server http.Server
}

// NewServer creates a new Server instance
func NewServer(port int) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", port),
			Handler: mux,
		}}

	mux.HandleFunc("/hello", s.HelloHandler)

	return s
}

func (s *Server) HelloHandler(w http.ResponseWriter, r *http.Request) {
	response := "test"
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		panic(err)
	}
}

func (s *Server) HealthCheck() error {
	log.Info("Server health check.")
	return nil
}

func (s *Server) Shutdown() error {
	log.Info("Server shutting down...")
	defer log.Info("Server shot down.")

	return s.server.Shutdown(context.Background())
}

// Run starts the HTTP server
func (s *Server) Run() error {
	log.Info("Starting server at: ", s.server.Addr)
	return s.server.ListenAndServe()
}
