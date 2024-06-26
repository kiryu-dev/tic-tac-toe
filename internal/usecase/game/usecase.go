package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/kiryu-dev/tic-tac-toe/pkg/utils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	maxReconnectionTime = 20 * time.Second
)

type useCase struct {
	mu     *sync.Mutex
	logger *zap.Logger
}

func New(logger *zap.Logger) useCase {
	return useCase{
		mu:     &sync.Mutex{},
		logger: logger,
	}
}

func (u useCase) Play(_ context.Context, player domain.Player, state *domain.GameState) error {
	u.mu.Lock()
	u.logger.Info("start playing", zap.String("player uuid", player.Uuid()))
	state.Status = domain.InProgress
	state.ActivePlayerCount++
	if err := startGame(player, state.Board); err != nil {
		return errors.WithMessage(err, "start game")
	}
	u.mu.Unlock()

	go u.handleEnemyReconnect(player, state)

	switch {
	case state.RecoveredPlayer == player.Uuid():
		if err := player.SendMessage(domain.Message{Type: domain.RequestMove}); err != nil {
			return errors.WithMessage(err, "send message to player")
		}
		isGameFinished, err := u.handlePlayersMove(player, state)
		if err != nil {
			return errors.WithMessage(err, "handle player's move")
		}
		if isGameFinished {
			return nil
		}
	case state.Round == 0 && player.Cell() == domain.O:
		player.MakeMove(domain.Move{Status: domain.NoneMove})
	}

	for {
		select {
		case v := <-player.GetEnemyMove():
			switch v.Status {
			case domain.NoneMove:
				if err := player.SendMessage(domain.Message{Type: domain.RequestMove}); err != nil {
					return errors.WithMessage(err, "send message to player")
				}
			case domain.MoveX, domain.MoveO:
				err := sendMoveMessage(player, v.Position, domain.WithCellType(v.CellType), domain.RequestMoveBack())
				if err != nil {
					return errors.WithMessage(err, "send message")
				}
			case domain.Disconnect:
				if err := u.handleEnemyDisconnect(player, state); err != nil {
					return errors.WithMessage(err, "handle enemy disconnect")
				}
				return nil
			default:
				u.mu.Lock()
				state.Status = domain.Finished
				u.mu.Unlock()

				gameResult, err := toGameResult(v.Status, player)
				if err != nil {
					return errors.WithMessage(err, "to game result")
				}
				err = sendMoveMessage(player, v.Position, domain.WithCellType(v.CellType), domain.WithGameResult(gameResult))
				if err != nil {
					return errors.WithMessage(err, "send move message")
				}
				return nil
			}

			isGameFinished, err := u.handlePlayersMove(player, state)
			if err != nil {
				return errors.WithMessage(err, "handle player's move")
			}
			if isGameFinished {
				return nil
			}
		}
	}
}

func (u useCase) handlePlayersMove(player domain.Player, state *domain.GameState) (isGameFinished bool, err error) {
	move, err := receiveMoveMessage(player, state.Board)
	switch {
	case errors.Is(err, domain.ErrConnectionClosed):
		player.MakeMove(domain.Move{Status: domain.Disconnect})
		return true, nil
	case err != nil:
		return false, errors.WithMessage(err, "receive move message")
	}

	moveStatus, err := u.executeMove(move, state)
	if err != nil {
		return false, errors.WithMessage(err, "execute player's move")
	}
	//printBoard(state.Board)

	defer func() {
		if err != nil {
			player.MakeMove(domain.Move{Status: domain.Disconnect})
			return
		}
		player.MakeMove(domain.Move{
			CellType: player.Cell(),
			Position: move.Position,
			Status:   moveStatus,
		})
	}()

	//player.MakeMove(domain.Move{
	//	CellType: player.Cell(),
	//	Position: move.Position,
	//	Status:   moveStatus,
	//})

	gameResult, err := toGameResult(moveStatus, player)
	switch {
	case errors.Is(err, errUnexpectedMoveStatus):
		/* the game isn't over, it's still in progress */
		if err := sendMoveMessage(player, move.Position); err != nil {
			return false, errors.WithMessage(err, "send move message")
		}
		return false, nil
	case err != nil:
		return false, errors.WithMessage(err, "to game result") /* impossible case but.... */
	default:
		err := sendMoveMessage(player, move.Position, domain.WithGameResult(gameResult))
		if err != nil {
			return false, errors.WithMessage(err, "send move message")
		}
		return true, nil
	}
}

func (u useCase) handleEnemyDisconnect(player domain.Player, state *domain.GameState) error {
	u.mu.Lock()
	state.Status = domain.Finished
	u.mu.Unlock()

	err := player.SendMessage(domain.Message{
		Type:    domain.Walkover,
		Payload: domain.WalkoverPayload{GameResult: WalkoverGameResult},
	})
	if err != nil {
		return errors.WithMessage(err, "send message to player")
	}
	return nil
}

func startGame(player domain.Player, board domain.Board) error {
	err := player.SendMessage(domain.Message{
		Type: domain.StartGame,
		Payload: domain.StartGamePayload{
			CellType: player.Cell(),
			Board:    board,
		},
	})
	if err != nil {
		return errors.WithMessage(err, "send message to player")
	}
	return nil
}

func sendMoveMessage(player domain.Player, position byte, opts ...domain.PlayerMovePayloadOption) error {
	payload := &domain.PlayerMovePayload{
		CellType: player.Cell(),
		Position: position,
	}
	for _, opt := range opts {
		opt(payload)
	}
	err := player.SendMessage(domain.Message{
		Type:    domain.PlayerMove,
		Payload: payload,
	})
	if err != nil {
		return errors.WithMessage(err, "send message to player")
	}
	return nil
}

func receiveMoveMessage(player domain.Player, board domain.Board) (domain.PlayerMovePayload, error) {
	for {
		msg, err := player.ReceiveMessage()
		if err != nil {
			return domain.PlayerMovePayload{}, errors.WithMessage(err, "read message from player")
		}
		if msg.Type != domain.PlayerMove {
			return domain.PlayerMovePayload{}, errors.New("unexpected message type")
		}

		move, err := utils.UnmarshalJson[domain.PlayerMovePayload](msg.Payload)
		if err != nil {
			return domain.PlayerMovePayload{}, errors.WithMessage(err, "unmarshal player's move")
		}
		if player.Cell() != move.CellType {
			return domain.PlayerMovePayload{}, errors.Errorf(
				"expected cell type '%c', got '%c'", player.Cell(), move.CellType,
			)
		}

		err = validateMovePosition(board, move.Position)
		switch {
		case errors.Is(err, errInvalidSelectedPosition):
			if err := player.SendMessage(domain.Message{Type: domain.RequestMove}); err != nil {
				return domain.PlayerMovePayload{}, errors.WithMessage(err, "send message to player")
			}
		case err != nil:
			return domain.PlayerMovePayload{}, errors.WithMessage(err, "validate player's move")
		default:
			return move, nil
		}
	}
}

func validateMovePosition(board domain.Board, pos byte) error {
	if pos > 8 {
		return errInvalidSelectedPosition
	}
	if board[pos] != domain.None {
		return errors.WithMessagef(errInvalidSelectedPosition,
			"cell in position '%d' is already selected", pos)
	}
	return nil
}

const maxRounds = 9

var winConditions = [8][3]uint8{
	{0, 1, 2},
	{3, 4, 5},
	{6, 7, 8},
	{0, 3, 6},
	{1, 4, 7},
	{2, 5, 8},
	{0, 4, 8},
	{2, 4, 6},
}

func (u useCase) executeMove(move domain.PlayerMovePayload, state *domain.GameState) (domain.MoveStatus, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	state.Board[move.Position] = move.CellType
	state.Round++
	state.CurrentMove = invertCellType(move.CellType)
	if isWinnable(state.Board, move.CellType) {
		switch move.CellType {
		case domain.X:
			return domain.WinX, nil
		case domain.O:
			return domain.WinO, nil
		default:
			return domain.NoneMove, errors.New("unexpected cell type")
		}
	}
	if state.Round == maxRounds {
		return domain.Draw, nil
	}
	switch move.CellType {
	case domain.X:
		return domain.MoveX, nil
	case domain.O:
		return domain.MoveO, nil
	default:
		return domain.NoneMove, errors.New("unexpected cell type")
	}
}

func invertCellType(cellType domain.Cell) domain.Cell {
	switch cellType {
	case domain.X:
		return domain.O
	case domain.O:
		return domain.X
	default:
		return cellType
	}
}

func isWinnable(board domain.Board, cellType domain.Cell) bool {
	for _, condition := range winConditions {
		isWinnable := true
		for _, v := range condition {
			if board[v] != cellType {
				isWinnable = false
				break
			}
		}
		if isWinnable {
			return true
		}
	}
	return false
}

const (
	WinGameResult      = "Победа"
	LoseGameResult     = "Поражение"
	DrawGameResult     = "Ничья"
	WalkoverGameResult = "Техническая победа (оппонент отключился)"
)

func toGameResult(status domain.MoveStatus, player domain.Player) (string, error) {
	winnerCell := domain.O
	switch status {
	case domain.WinX:
		winnerCell = domain.X
		fallthrough
	case domain.WinO:
		if player.Cell() != winnerCell {
			return LoseGameResult, nil
		}
		return WinGameResult, nil
	case domain.Draw:
		return DrawGameResult, nil
	case domain.Disconnect:
		return WalkoverGameResult, nil
	default:
		return "", errUnexpectedMoveStatus
	}
}

func (u useCase) handleEnemyReconnect(player domain.Player, state *domain.GameState) {
	select {
	case <-time.After(maxReconnectionTime):
		if state.ActivePlayerCount != 2 {
			if err := u.handleEnemyDisconnect(player, state); err != nil {
				u.logger.Warn(err.Error())
			}
		}
	}
}

func printBoard(board domain.Board) {
	for i, v := range board {
		fmt.Printf("%c ", v)
		if (i+1)%3 == 0 {
			fmt.Println()
		}
	}
}
