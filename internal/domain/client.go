package domain

import (
	"github.com/pkg/errors"
)

var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrEmptyMessage     = errors.New("empty message")
)

const (
	ClientUuidHeader = "X-Client-Key"
)

type messageType byte

const (
	StartGame = messageType(iota)
	RequestMove
	PlayerMove
	Walkover
	SwitchServer
)

type Message struct {
	Type    messageType
	Payload any
}

type StartGamePayload struct {
	CellType Cell
	Board    Board
}

type PlayerMovePayload struct {
	CellType        Cell
	Position        byte
	IsMoveRequested bool
	GameResult      *string
}

type WalkoverPayload struct {
	GameResult string
}

type SwitchServerPayload struct {
	MasterServer string
}

type PlayerMovePayloadOption func(p *PlayerMovePayload)

func RequestMoveBack() PlayerMovePayloadOption {
	return func(p *PlayerMovePayload) {
		p.IsMoveRequested = true
	}
}

func WithGameResult(gameResultMsg string) PlayerMovePayloadOption {
	return func(p *PlayerMovePayload) {
		p.GameResult = &gameResultMsg
	}
}

func WithCellType(cellType Cell) PlayerMovePayloadOption {
	return func(p *PlayerMovePayload) {
		p.CellType = cellType
	}
}

type Client interface {
	WriteMessage(msg Message) error
	ReadMessage() (Message, error)
	Uuid() string
}
