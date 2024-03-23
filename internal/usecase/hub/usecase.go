package hub

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	clientQueueBufSize = 2
)

type enqueuedClient struct {
	client     domain.Client
	resultChan chan domain.Player
}

type useCase struct {
	game        domain.GameUseCase
	clientQueue chan enqueuedClient
	gamesStates map[string]*domain.GameState
	mu          *sync.RWMutex
	logger      *zap.Logger
}

func New(game domain.GameUseCase, logger *zap.Logger) useCase {
	u := useCase{
		game:        game,
		clientQueue: make(chan enqueuedClient, clientQueueBufSize),
		gamesStates: make(map[string]*domain.GameState),
		mu:          &sync.RWMutex{},
		logger:      logger,
	}
	go u.createGames()
	return u
}

func (u useCase) Handle(ctx context.Context, client domain.Client) error {
	player := u.enqueueForGame(client)
	u.mu.RLock()
	gameState := u.gamesStates[player.GameUuid()]
	u.mu.RUnlock()
	if err := u.game.Play(ctx, player, gameState); err != nil {
		return errors.WithMessage(err, "play game")
	}
	return nil
}

func (u useCase) enqueueForGame(client domain.Client) domain.Player {
	ch := make(chan domain.Player)
	defer close(ch)
	u.clientQueue <- enqueuedClient{
		client:     client,
		resultChan: ch,
	}
	return <-ch
}

func (u useCase) createGames() {
	for {
		for len(u.clientQueue) > 1 {
			gameUuid, playerX, playerO := u.createGame()
			lhs := <-u.clientQueue
			rhs := <-u.clientQueue
			moveChan := make(chan domain.Move)
			lhs.resultChan <- domain.NewPlayer(playerX, gameUuid, lhs.client, domain.X, moveChan)
			rhs.resultChan <- domain.NewPlayer(playerO, gameUuid, rhs.client, domain.O, moveChan)
		}
	}
}

func (u useCase) createGame() (string, string, string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	var (
		gameUuid = uuid.NewString()
		playerX  = uuid.NewString()
		playerO  = uuid.NewString()
	)
	var board domain.Board
	for i := range board {
		board[i] = domain.None
	}
	u.gamesStates[gameUuid] = &domain.GameState{
		Board:       board,
		PlayerX:     playerX,
		PlayerO:     playerO,
		CurrentMove: domain.X,
		Status:      domain.ReadyToStart,
	}
	return gameUuid, playerX, playerO
}
