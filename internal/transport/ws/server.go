package ws

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"go.uber.org/zap"
)

type server struct {
	srv        *http.Server
	hub        domain.HubUseCase
	sync       domain.SyncUseCase
	role       domain.ServerRole
	masterHost string
	upgrader   websocket.Upgrader
	logger     *zap.Logger
	done       chan struct{}
}

func New(hub domain.HubUseCase, sync domain.SyncUseCase, logger *zap.Logger) *server {
	return &server{
		srv:  &http.Server{Addr: os.Getenv("SERVER_PORT")},
		hub:  hub,
		sync: sync,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Пропускаем любой запрос
			},
		},
		logger: logger,
		done:   make(chan struct{}),
	}
}

func (s *server) ListenAndServe(ctx context.Context) {
	s.initRoutes()
	go func() {
		s.logger.Info("starting listening address: " + s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil {
			s.logger.Info(err.Error())
		}
	}()
	time.Sleep(3 * time.Second)
	go s.sync.Sync(ctx, s.hub.GamesStates())
	go s.sync.DefineMasterServer(ctx)
	for {
		select {
		case info := <-s.sync.Chan():
			s.logger.Info("server info", zap.Any("info", info))
			s.masterHost = info.MasterServerName
			s.role = info.ServerRole
			if err := s.sync.CheckMasterHealth(ctx); err != nil {
				s.logger.Error(err.Error())
			}
		case <-s.done:
			return
		}
	}
}

func (s *server) Shutdown() error {
	s.done <- struct{}{}
	return s.srv.Shutdown(context.Background())
}

func (s *server) initRoutes() {
	http.HandleFunc("/game", s.serveWs)
	http.HandleFunc("GET /health", s.healthCheck)
	http.HandleFunc("POST /sync", s.applyStates)
}
