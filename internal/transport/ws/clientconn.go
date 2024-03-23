package ws

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/pkg/errors"
)

type client struct {
	conn *websocket.Conn
}

func newClient(conn *websocket.Conn) client {
	return client{conn: conn}
}

func (c client) WriteMessage(msg domain.Message) error {
	if err := c.conn.WriteJSON(msg); err != nil {
		return errors.WithMessage(err, "websocket conn write json")
	}
	return nil
}

func (c client) ReadMessage() (domain.Message, error) {
	var msg domain.Message
	if err := c.conn.ReadJSON(&msg); err != nil {
		return domain.Message{}, errors.WithMessage(err, "websocket conn read json")
	}
	fmt.Println(msg)
	return msg, nil
}

func (c client) Close() {
	_ = c.conn.Close()
}
