package wsserver

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/zhulik/fid/pkg/log"
)

var (
	wsLogger = log.Logger.WithField("component", "wsserver.WebsocketConnection")
)

const (
	PingInterval = time.Second * 10
	PingTimeout  = time.Second * 2
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

func (w *WebSocketConnection) WriteRead(payload string) (string, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, []byte(payload))
	if err != nil {
		return "", err
	}

	_, message, err := w.conn.ReadMessage()
	if err != nil {
		return "", err
	}

	return string(message), nil
}

func (w *WebSocketConnection) Handle() error {
	defer w.Close()

	wsLogger.Info("Function '", w.name, "' handler is being handled...")
	defer wsLogger.Info("Function '", w.name, "' handler is not being handled anymore.")

	w.ping()

	return nil
}

func (w *WebSocketConnection) Close() error {
	return w.conn.Close()
}

func (w *WebSocketConnection) ping() {
	for {
		time.Sleep(PingInterval)
		wsLogger.Debug("Pinging function '", w.name, "'...")

		err := w.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(PingTimeout))
		if err != nil {
			// TODO: exit silently when already disconnected.
			wsLogger.Debug("Function '", w.name, "' ping error, closing connection...")
			w.Close()
		}

		wsLogger.Debug("Pong received from function '", w.name, "'...")
	}
}
