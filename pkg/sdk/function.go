package sdk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type ContextKey int

const (
	RequestID ContextKey = iota
	// ...
)

const (
	ResponseTimeout = 5 * time.Second
)

var (
	// TODO: extract constants.
	apiURL  = os.Getenv("AWS_LAMBDA_RUNTIME_API")                                 //nolint:gochecknoglobals
	nextURL = fmt.Sprintf("http://%s/2018-06-01/runtime/invocation/next", apiURL) //nolint:gochecknoglobals

	// TODO: use jwt instead.
	functionName = os.Getenv("FID_FUNCTION_NAME") //nolint:gochecknoglobals

	ErrUnexpectedStatus    = errors.New("unexpected status code")
	ErrCannotParseDeadline = errors.New("cannot parse deadline")
)

type Handler func(ctx context.Context, req []byte) ([]byte, error)

func port() int {
	port := 80

	portStr := os.Getenv("HTTP_PORT")
	if portStr != "" {
		var err error

		port, err = strconv.Atoi(portStr)
		if err != nil {
			panic(err)
		}
	}

	return port
}

func Serve(handler Handler) error {
	go server(handler)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, nextURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Function-Name", functionName)

	for {
		err := fetchEventAndHandle(req, handler)
		if err != nil {
			return err
		}
	}
}

func fetchEventAndHandle(nextReq *http.Request, handler Handler) error {
	// TODO: recover from panic in the handler
	resp, err := fetchNextEvent(nextReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	deadline, err := parseDeadline(resp)
	if err != nil {
		return err
	}

	event, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	var respReq *http.Request

	var reqErr error

	requestID := resp.Header.Get("Lambda-Runtime-Aws-Request-Id")

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	ctx = context.WithValue(ctx, RequestID, requestID)

	result, err := handler(ctx, event)
	if err != nil {
		// TODO: properly serialize error
		respReq, reqErr = http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("http://%s/2018-06-01/runtime/invocation/%s/error", apiURL, requestID),
			bytes.NewReader(result),
		)
	} else {
		respReq, reqErr = http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("http://%s/2018-06-01/runtime/invocation/%s/response", apiURL, requestID),
			bytes.NewReader(result),
		)
	}

	if reqErr != nil {
		return fmt.Errorf("failed to create response request: %w", err)
	}

	err = postResponse(respReq)
	if err != nil {
		return err
	}

	return nil
}

func fetchNextEvent(nextReq *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(nextReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get next event: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %d, failed to read body", ErrUnexpectedStatus, resp.StatusCode)
		}

		return nil, fmt.Errorf("%w: %d, body=%s", ErrUnexpectedStatus, resp.StatusCode, body)
	}

	return resp, nil
}

func postResponse(respReq *http.Request) error {
	respCtx, cancel := context.WithTimeout(context.Background(), ResponseTimeout)
	defer cancel()

	respReq = respReq.WithContext(respCtx)

	resp, err := http.DefaultClient.Do(respReq)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	defer resp.Body.Close()

	return nil
}

func parseDeadline(resp *http.Response) (time.Time, error) {
	deadlineStr := resp.Header.Get("Lambda-Runtime-Deadline-Ms")
	deadline, ok := strconv.ParseInt(deadlineStr, 10, 64)

	if ok != nil {
		return time.Time{}, fmt.Errorf("%w: '%s'", ErrCannotParseDeadline, deadlineStr)
	}

	return time.UnixMilli(deadline), nil
}

func server(handler Handler) {
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		log.Printf("healthcheck")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/invoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}
		defer r.Body.Close()
		log.Printf("Invoking function...")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		resp, err := handler(r.Context(), body)
		if err != nil {
			panic(err)
		}

		w.WriteHeader(http.StatusOK)

		_, err = w.Write(resp)
		if err != nil {
			panic(err)
		}
	})

	addr := fmt.Sprintf(":%d", port())
	log.Printf("Starting health check http server at: %s", addr)

	err := http.ListenAndServe(addr, nil) //nolint:gosec
	if err != nil {
		panic(err)
	}
}
