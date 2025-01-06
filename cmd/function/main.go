package main

import (
	"github.com/gorilla/websocket"
	"log"
	"os"
)

func handler(input []byte) ([]byte, error) {
	log.Printf("Handling %s:", string(input))
	return []byte("test"), nil
}

func main() {
	// Connect to the WebSocket server
	serverURL := os.Getenv("WS_URI")
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			panic(err)
		}

		result, err := handler(msg)
		if err != nil {
			panic(err)
		}

		err = conn.WriteMessage(websocket.TextMessage, result)
		if err != nil {
			panic(err)
		}
	}
}
