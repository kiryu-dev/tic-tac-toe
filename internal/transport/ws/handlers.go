package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"go.uber.org/zap"
)

func serveWs(upgrader websocket.Upgrader, hub domain.HubUseCase, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		client := newClient(conn)
		defer client.Close()
		if err := hub.Handle(r.Context(), client); err != nil {
			logger.Error(err.Error())
		}
	}
}
