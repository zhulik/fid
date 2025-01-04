package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/samber/do"
	"net/http"

	"github.com/zhulik/fid/pkg/log"
)

type Server struct {
	injector *do.Injector
	server   http.Server
}

// NewServer creates a new Server instance
func NewServer(injector *do.Injector) *Server {
	mux := http.NewServeMux()

	s := &Server{
		injector: injector,
		server: http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:8080"), // TODO: read port from config
			Handler: mux,
		}}

	mux.HandleFunc("/hello", LoggingMiddleware(s.HelloHandler))

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
