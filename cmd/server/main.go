package main

import (
	"github.com/kiryu-dev/tic-tac-toe/internal/transport/ws"
	"github.com/kiryu-dev/tic-tac-toe/internal/usecase/game"
	"github.com/kiryu-dev/tic-tac-toe/internal/usecase/hub"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()
	game := game.New(logger)
	hub := hub.New(game, logger)
	server := ws.New(":8080", hub, logger)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal(err.Error())
	}
}
