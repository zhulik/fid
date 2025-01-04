package httpserver

import (
	"errors"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/log"
	"net/http"
)

func Register(injector *do.Injector) {
	do.Provide(injector, func(injector *do.Injector) (*Server, error) {
		server := NewServer(8080)

		go func() {
			err := server.Run()

			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			log.Error(err)
		}()
		return server, nil
	})
}
