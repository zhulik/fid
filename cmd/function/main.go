package main

import (
	"context"
	"log"

	"github.com/zhulik/fid/pkg/sdk"
)

func handler(_ context.Context, input []byte) ([]byte, error) { //nolint:unparam
	log.Printf("Handling %s:", string(input))

	return []byte("test"), nil
}

func main() {

	sdk.Serve(handler)
}
