package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/kiryu-dev/tic-tac-toe/internal/adapters/webapi"
	"github.com/kiryu-dev/tic-tac-toe/internal/config"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/kiryu-dev/tic-tac-toe/internal/transport/ws"
	"github.com/kiryu-dev/tic-tac-toe/internal/usecase/game"
	"github.com/kiryu-dev/tic-tac-toe/internal/usecase/hub"
	"github.com/kiryu-dev/tic-tac-toe/internal/usecase/synchronizer"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()
	cfgPath := flag.String("config", "./config.yml", "path to config")
	flag.Parse()
	cfg, err := config.New(*cfgPath)
	if err != nil {
		logger.Fatal(err.Error())
	}
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	errGroup := new(errgroup.Group)
	errGroup.Go(func() error {
		select {
		case s := <-sigChan:
			return errors.Errorf("captured signal: %v", s)
		}
	})
	statesChan := make(chan map[string]*domain.GameState)
	defer close(statesChan)
	var (
		repo   = webapi.New()
		sync   = synchronizer.New(repo, statesChan, cfg.Servers, logger)
		game   = game.New(logger)
		hub    = hub.New(game, statesChan, logger)
		server = ws.New(hub, sync, logger)
	)
	go server.ListenAndServe(context.Background())
	if err := errGroup.Wait(); err != nil {
		logger.Info("gracefully shutting down the server: " + err.Error())
	}
	if err := server.Shutdown(); err != nil {
		logger.Info("failed to shutdown http server: " + err.Error())
	}
}
