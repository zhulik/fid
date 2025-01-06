package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	// Connect to the WebSocket server
	serverURL := os.Getenv("WS_URI")
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)

	// Start a goroutine to read messages from the WebSocket connection
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			log.Printf("Received: %s", message)
		}
	}()

	// Set up signal handling to gracefully shut down the client
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Send a message to the server every second
	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			message := fmt.Sprintf("Hello %s", t)
			log.Printf("Sending: %s", message)
			err := conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Println("Write error:", err)
				return
			}
		case <-interrupt:
			log.Println("Interrupt received, shutting down...")
			// Gracefully close the connection
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Close error:", err)
				return
			}
			return
		}
	}
}
