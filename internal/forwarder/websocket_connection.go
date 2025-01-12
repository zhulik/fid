package forwarder

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
		conn: conn,
		name: name,
		logger: logger.WithFields(logrus.Fields{
			"function":  name,
			"component": "websocket-connection",
		}),
	}
}

func (w *WebSocketConnection) WriteRead(payload string) (string, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, []byte(payload))
	if err != nil {
		return "", fmt.Errorf("failed to write WS message: %w", err)
	}

	_, message, err := w.conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("failed to read WS message: %w", err)
	}

	return string(message), nil
}

func (w *WebSocketConnection) Handle() error {
	defer w.Close()

	w.logger.Info("Function handler is being handled...")
	defer w.logger.Info("Function handler is not being handled anymore.")

	w.ping()

	return nil
}

func (w *WebSocketConnection) Close() error {
	err := w.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close WS connection: %w", err)
	}

	return nil
}

func (w *WebSocketConnection) ping() {
	for {
		time.Sleep(PingInterval)
		w.logger.Debug("Pinging function ...")

		err := w.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(PingTimeout))
		if err != nil {
			// TODO: exit silently when already disconnected.
			w.logger.Debug("Function ping error, closing connection...")
			w.Close()
		}

		w.logger.Debug("Pong received from function.")
	}
}
