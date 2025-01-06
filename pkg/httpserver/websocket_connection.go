package httpserver

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zhulik/fid/pkg/log"
)

var (
	wsLogger = log.Logger.WithField("component", "httpserver.WebsocketConnection")

	PingInterval = time.Second * 5
)

type WebSocketConnection struct {
	conn *websocket.Conn
	name string
}

func NewWebsocketConnection(name string, conn *websocket.Conn) *WebSocketConnection {
	return &WebSocketConnection{
		conn: conn,
		name: name,
	}
}

func (w *WebSocketConnection) Handle() error {
	defer w.Close()

	wsLogger.Info("Function '", w.name, "' handler is being handled...")
	defer wsLogger.Info("Function '", w.name, "' handler is not being handled anymore.")

	go w.ping()

	for {
		messageType, message, err := w.conn.ReadMessage()
		if err != nil {
			return err
		}

		fmt.Printf("Received: %s\n", message)
		if err := w.conn.WriteMessage(messageType, message); err != nil {
			return err
		}
	}
}

func (w *WebSocketConnection) Close() error {
	return w.conn.Close()
}

func (w *WebSocketConnection) ping() {
	for {
		time.Sleep(PingInterval)
		wsLogger.Info("Pinging function '", w.name, "'...")
		err := w.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
		if err != nil {
			// TODO: exit silently when already disconnected.
			wsLogger.Info("Function '", w.name, "' ping error, closing connection...")
			w.Close()
			return
		}
		wsLogger.Info("Pong received from function '", w.name, "'...")
	}
}
