package httpserver

import (
	"context"
	"fmt"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zhulik/fid/pkg/log"
)

var (
	logger = log.Logger.WithField("component", "httpserver.Server")
)

type Server struct {
	injector *do.Injector
	backend  core.Backend
	server   http.Server
	error    error
}

// NewServer creates a new Server instance
func NewServer(injector *do.Injector) (*Server, error) {
	logger.Info("Creating new server...")
	defer logger.Info("Server created.")

	router := mux.NewRouter()
	router.Use(Middlewares)

	backend, err := do.Invoke[core.Backend](injector)
	if err != nil {
		return nil, err
	}

	s := &Server{
		injector: injector,
		backend:  backend,
		server: http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:8080"), // TODO: read port from config
			Handler: router,
		},
	}

	router.HandleFunc("/info", s.InfoHandler).Methods("GET")

	router.HandleFunc("/pulse", s.PulseHandler).Methods("GET")
	router.HandleFunc("/invoke/{functionName}", s.InvokeHandler).Methods("POST")

	router.HandleFunc("/", s.NotFoundHandler)

	return s, nil
}

func (s *Server) InfoHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	info, err := s.backend.Info(r.Context())
	if err != nil {
		panic(err)
	}

	err = WriteJSON(info, w)
	if err != nil {
		panic(err)
	}
}

func (s *Server) PulseHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	errs := s.injector.HealthCheck()

	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

func (s *Server) InvokeHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	name := vars["functionName"]
	logger.Info("Invoking ", name, "...")
}

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.WriteHeader(http.StatusNotFound)
	err := WriteJSON(ErrorBody{
		Error: "Not found",
	}, w)
	if err != nil {
		panic(err)
	}
}

func (s *Server) HealthCheck() error {
	logger.Info("Server health check.")
	return s.error
}

func (s *Server) Shutdown() error {
	logger.Info("Server shutting down...")
	defer logger.Info("Server shot down.")

	return s.server.Shutdown(context.Background())
}

// Run starts the HTTP server
func (s *Server) Run() error {
	logger.Info("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()
	return s.error
}
