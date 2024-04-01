package hub

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	clientQueueBufSize = 2
	syncPeriod         = 5 * time.Second
)

type enqueuedClient struct {
	client     domain.Client
	resultChan chan domain.Player
}

type useCase struct {
	game        domain.GameUseCase
	clientQueue chan enqueuedClient
	gamesStates map[string]*domain.GameState
	statesChan  chan map[string]*domain.GameState
	ticker      *time.Ticker
	mu          *sync.RWMutex
	logger      *zap.Logger
}

func New(game domain.GameUseCase, logger *zap.Logger) *useCase {
	u := &useCase{
		game:        game,
		clientQueue: make(chan enqueuedClient, clientQueueBufSize),
		gamesStates: make(map[string]*domain.GameState),
		statesChan:  make(chan map[string]*domain.GameState),
		ticker:      time.NewTicker(syncPeriod),
		mu:          &sync.RWMutex{},
		logger:      logger,
	}
	go u.createGames()
	go u.syncStates()
	return u
}

func (u *useCase) Handle(ctx context.Context, client domain.Client) error {
	player, ok := u.continueActiveGame(client)
	if !ok {
		player = u.enqueueForGame(client)
	}
	u.mu.RLock()
	gameState := u.gamesStates[player.GameUuid()]
	u.mu.RUnlock()
	if err := u.game.Play(ctx, player, gameState); err != nil {
		return errors.WithMessage(err, "play game")
	}
	return nil
}

func (u *useCase) enqueueForGame(client domain.Client) domain.Player {
	ch := make(chan domain.Player)
	defer close(ch)
	u.clientQueue <- enqueuedClient{
		client:     client,
		resultChan: ch,
	}
	return <-ch
}

func (u *useCase) createGames() {
	for {
		for len(u.clientQueue) > 1 {
			lhs := <-u.clientQueue
			rhs := <-u.clientQueue
			gameUuid := u.createGame(lhs.client.Uuid(), rhs.client.Uuid())
			moveChan := make(chan domain.Move)
			lhs.resultChan <- domain.NewPlayer(gameUuid, lhs.client, domain.X, moveChan)
			rhs.resultChan <- domain.NewPlayer(gameUuid, rhs.client, domain.O, moveChan)
		}
	}
}

func (u *useCase) createGame(playerX string, playerO string) string {
	u.mu.Lock()
	defer u.mu.Unlock()
	gameUuid := uuid.NewString()
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
	return gameUuid
}

func (u *useCase) syncStates() {
	defer u.ticker.Stop()
	for range u.ticker.C {
		currentGameCount := u.removeFinishedGames()
		if currentGameCount > 0 {
			u.statesChan <- u.gamesStates
		}
	}
}

func (u *useCase) GamesStates() <-chan map[string]*domain.GameState {
	return u.statesChan
}

func (u *useCase) removeFinishedGames() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	for gameUuid, state := range u.gamesStates {
		if state.Status == domain.Finished {
			delete(u.gamesStates, gameUuid)
		}
	}
	return len(u.gamesStates)
}

func (u *useCase) ApplyStates(_ context.Context, states map[string]*domain.GameState) {
	u.mu.Lock()
	u.gamesStates = states
	u.logger.Info("applied states", zap.Any("states", u.gamesStates))
	u.mu.Unlock()
}

func (u *useCase) continueActiveGame(client domain.Client) (domain.Player, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.logger.Info("trying to find active game with this client...", zap.Any("states", u.gamesStates))
	clientUuid := client.Uuid()
	for gameUuid, state := range u.gamesStates {
		if state.Status == domain.Finished {
			continue
		}
		cellType := domain.None
		switch clientUuid {
		case state.PlayerX:
			cellType = domain.X
		case state.PlayerO:
			cellType = domain.O
		default:
			continue
		}
		//if cellType == domain.None {
		//	continue
		//}
		if state.MoveChan == nil {
			state.MoveChan = make(chan domain.Move)
		}

		if state.RecoveredPlayer == "" && cellType == state.CurrentMove {
			state.RecoveredPlayer = clientUuid
		}

		u.logger.Info("found active game", zap.String("game uuid", gameUuid))
		return domain.NewPlayer(gameUuid, client, cellType, state.MoveChan), true
	}
	return domain.Player{}, false
}
