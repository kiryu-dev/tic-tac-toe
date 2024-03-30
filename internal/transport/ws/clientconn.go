package ws

import (
	"github.com/gorilla/websocket"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/pkg/errors"
)

type client struct {
	conn *websocket.Conn
	uuid string
}

func newClient(conn *websocket.Conn, uuid string) client {
	return client{
		conn: conn,
		uuid: uuid,
	}
}

func (c client) WriteMessage(msg domain.Message) error {
	if err := c.conn.WriteJSON(msg); err != nil {
		return errors.WithMessage(err, "websocket conn write json")
	}
	return nil
}

func (c client) ReadMessage() (domain.Message, error) {
	var msg domain.Message
	err := c.conn.ReadJSON(&msg)
	switch {
	case websocket.IsUnexpectedCloseError(err):
		return domain.Message{}, domain.ErrConnectionClosed
	case err != nil:
		return domain.Message{}, errors.WithMessage(err, "websocket conn read json")
	}
	return msg, nil
}

func (c client) Uuid() string {
	return c.uuid
}

func (c client) Close() {
	_ = c.conn.Close()
}
