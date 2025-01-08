package sdk

import (
	"context"
	"fmt"
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

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}

	defer conn.Close()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("healthcheck")
		w.WriteHeader(http.StatusOK)
	})
	go func() {
		addr := fmt.Sprintf(":%d", port())
		log.Printf("Starting health check http server at: %s", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			panic(err)
		}
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
