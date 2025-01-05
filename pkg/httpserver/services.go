package httpserver

import (
	"errors"
	"github.com/samber/do"
	"net/http"
)

func Register(injector *do.Injector) {
	do.Provide(injector, func(injector *do.Injector) (*Server, error) {
		server := NewServer(injector)

		go func() {
			err := server.Run()

			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			panic(err)
		}()
		return server, nil
	})
}
