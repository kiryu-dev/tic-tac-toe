package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"go.uber.org/zap"
)

func New(port string, hub domain.HubUseCase, logger *zap.Logger) *http.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Пропускаем любой запрос
		},
	}
	http.HandleFunc("/", serveWs(upgrader, hub, logger))
	return &http.Server{
		Addr:         port,
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}
}
