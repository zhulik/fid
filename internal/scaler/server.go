package scaler

import (
	"context"
	"fmt"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	scaler *Scaler
}

// NewServer creates a new Server instance.
func NewServer(ctx context.Context, injector *do.Injector) (*Server, error) {
	config := do.MustInvoke[core.Config](injector)
	logger := do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "scaler.Server")
	functionsRepo := do.MustInvoke[core.FunctionsRepo](injector)

	server, err := httpserver.NewServer(injector, logger, config.HTTPPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	function, err := functionsRepo.Get(ctx, config.FunctionName())
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	scaler, err := NewScaler(ctx, injector, function)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		Server: server,
		scaler: scaler,
	}

	return srv, nil
}

func (s *Server) Run() error {
	errs := make(chan error, 2) //nolint:mnd

	go func() {
		errs <- s.scaler.Run()
	}()

	go func() {
		errs <- s.Server.Run()
	}()

	return <-errs
}
