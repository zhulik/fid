package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/zhulik/fid/pkg/json"
	"github.com/zhulik/fid/pkg/sdk"
)

func cpuIntensiveCalculations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_ = rand.Float64() * rand.Float64() //nolint:gosec
		}
	}
}

var ErrTestError = errors.New("test error")

type Request struct {
	Sleep        int
	Panic        bool
	Error        bool
	Calculations int
}

type Response struct {
	Message string
}

func handler(ctx context.Context, input []byte) ([]byte, error) {
	requestID := ctx.Value(sdk.RequestID).(string) //nolint:forcetypeassert

	log.Printf("Handling request %s, input %s:", requestID, string(input))

	request, err := json.Unmarshal[Request](input)
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Duration(request.Sleep) * time.Second)

	if request.Panic {
		panic("panic")
	}

	if request.Error {
		return nil, ErrTestError
	}

	if request.Calculations != 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(request.Calculations)*time.Second)
		cpuIntensiveCalculations(ctx)
		cancel()
	}

	response := Response{
		Message: fmt.Sprintf("Event %s handled successfully", requestID),
	}

	return json.Marshal(response) //nolint:wrapcheck
}

func main() {
	if err := sdk.Serve(handler); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
