package sdk

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/websocket"
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

func Serve(handler Handler) {
	// Connect to the WebSocket server
	serverURL := os.Getenv("WS_URI")

	conn, response, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}

	defer response.Body.Close()
	defer conn.Close()

	go func() {
		server(handler)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			panic(err)
		}

		result, err := handler(context.TODO(), msg)
		if err != nil {
			panic(err)
		}

		err = conn.WriteMessage(websocket.TextMessage, result)
		if err != nil {
			panic(err)
		}
	}
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
