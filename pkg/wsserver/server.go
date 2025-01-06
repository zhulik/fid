package wsserver

import (
	"context"
	"fmt"
	"github.com/zhulik/fid/pkg/httpserver"
	"net/http"

	"github.com/samber/do"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/log"
)

var (
	logger = log.Logger.WithField("component", "wsserver.Server")

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type Server struct {
	injector *do.Injector
	backend  core.ContainerBackend
	server   http.Server
	error    error
}

// NewServer creates a new Server instance
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

	s := &Server{
		injector: injector,
		backend:  backend,
		server: http.Server{
			Addr:    fmt.Sprintf(fmt.Sprintf("0.0.0.0:%d", config.WSServerPort())),
			Handler: router,
		},
	}

	router.HandleFunc("/pulse", s.PulseHandler).Methods("GET").Name("pulse")
	// TODO: authentication
	router.HandleFunc("/ws/{functionName}", s.WebsocketHandler).Methods("GET").Name("ws")

	router.HandleFunc("/", s.NotFoundHandler)

	return s, nil
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

func (s *Server) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	functionName := vars["functionName"]
	//_, err := s.backend.Function(r.Context(), functionName)
	//if errors.Is(err, core.ErrFunctionNotFound) {
	//	err := WriteJSON(ErrorBody{Error: "Not found"}, w)
	//	if err != nil {
	//		panic(err)
	//	}
	//	return
	//}

	logger.Debug("Function '", functionName, "' handler connected, upgrading to websocket connection...")
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}

	logger.Debug("Function '", functionName, "' handler successfully upgraded to websocket connection")

	wsConn := NewWebsocketConnection(functionName, conn)
	// TODO: handle error?
	go wsConn.Handle()
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

// Run starts the HTTP server
func (s *Server) Run() error {
	logger.Info("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()
	return s.error
}
