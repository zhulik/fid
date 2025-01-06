package wsserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/httpserver"
	"github.com/zhulik/fid/pkg/log"
)

var (
	logger = log.Logger.WithField("component", "wsserver.Server")
)

type Server struct {
	injector *do.Injector
	backend  core.ContainerBackend
	server   http.Server
	error    error
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	logger.Info("Creating new server...")
	defer logger.Info("Server created.")

	router := mux.NewRouter()
	router.Use(httpserver.JSONMiddleware(logger))
	router.Use(httpserver.RecoverMiddleware(logger))
	router.Use(httpserver.LoggingMiddleware(logger))

	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		return nil, err
	}

	server := &Server{
		injector: injector,
		backend:  backend,
		server: http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%d", config.WSServerPort()),
			ReadHeaderTimeout: httpserver.ReadHeaderTimeout,
			Handler:           router,
		},
	}

	router.HandleFunc("/pulse", server.PulseHandler).Methods("GET").Name("pulse")
	// TODO: authentication
	router.HandleFunc("/ws/{functionName}", server.WebsocketHandler).Methods("GET").Name("ws")

	router.HandleFunc("/", server.NotFoundHandler)

	return server, nil
}

func (s *Server) PulseHandler(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	errs := s.injector.HealthCheck()

	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

func (s *Server) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	functionName := vars["functionName"]
	// _, err := s.backend.Function(r.Context(), functionName)
	// if errors.Is(err, core.ErrFunctionNotFound) {
	//	err := WriteJSON(ErrorBody{Error: "Not found"}, w)
	//	if err != nil {
	//		panic(err)
	//	}
	//	return
	//}

	logger.Debug("Function '", functionName, "' handler connected, upgrading to websocket connection...")
	// Upgrade the HTTP connection to a WebSocket connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}

	logger.Debug("Function '", functionName, "' handler successfully upgraded to websocket connection")

	wsConn := NewWebsocketConnection(functionName, conn)
	// TODO: handle error?
	go wsConn.Handle() //nolint:errcheck
}

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	err := httpserver.WriteJSON(httpserver.ErrorBody{
		Error: "Not found",
	}, w, http.StatusNotFound)
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

// Run starts the HTTP server.
func (s *Server) Run() error {
	logger.Info("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()

	return s.error
}
