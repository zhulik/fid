package infoserver

import (
	"context"

	"github.com/samber/do"
)

func Register(ctx context.Context) {
	do.Provide(nil, NewServer)
}
