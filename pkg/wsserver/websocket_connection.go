package wsserver

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

const (
	PingInterval = time.Second * 10
	PingTimeout  = time.Second * 2
)

type WebSocketConnection struct {
	conn *websocket.Conn
	name string

	logger logrus.FieldLogger
}

func NewWebsocketConnection(name string, conn *websocket.Conn, logger logrus.FieldLogger) *WebSocketConnection {
	return &WebSocketConnection{
		conn:   conn,
		name:   name,
		logger: logger.WithField("component", "websocket-connection"),
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

	w.logger.Info("Function '", w.name, "' handler is being handled...")
	defer w.logger.Info("Function '", w.name, "' handler is not being handled anymore.")

	w.ping()

	return nil
}

func (w *WebSocketConnection) Close() error {
	return w.conn.Close()
}

func (w *WebSocketConnection) ping() {
	for {
		time.Sleep(PingInterval)
		w.logger.Debug("Pinging function '", w.name, "'...")

		err := w.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(PingTimeout))
		if err != nil {
			// TODO: exit silently when already disconnected.
			w.logger.Debug("Function '", w.name, "' ping error, closing connection...")
			w.Close()
		}

		w.logger.Debug("Pong received from function '", w.name, "'...")
	}
}
