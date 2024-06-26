package domain

import (
	"context"
)

type Cell byte

const (
	None = Cell(' ')
	X    = Cell('X')
	O    = Cell('O')
)

type MoveStatus byte

const (
	NoneMove = MoveStatus(iota)
	MoveX
	MoveO
	Draw
	WinX
	WinO
	Disconnect
)

type Move struct {
	CellType Cell
	Position byte
	Status   MoveStatus
}

type Board [9]Cell

type status byte

const (
	ReadyToStart = status(iota)
	InProgress
	Finished
)

type GameState struct {
	Board             Board
	PlayerX           string
	PlayerO           string
	RecoveredPlayer   string `json:"-"` /* TODO: `RecoveredPlayer` такой себе нейминг.. другой бы.. */
	CurrentMove       Cell
	Status            status
	Round             uint8
	ActivePlayerCount uint8     `json:"-"`
	MoveChan          chan Move `json:"-"`
}

type GameUseCase interface {
	Play(ctx context.Context, player Player, state *GameState) error
}
