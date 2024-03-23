package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/kiryu-dev/tic-tac-toe/pkg/utils"
	"github.com/pkg/errors"
)

func main() {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), map[string][]string{})
	if err != nil {
		log.Fatal("dial: " + err.Error())
	}
	defer func() {
		_ = conn.Close()
	}()
	client := newClient(conn)
	if err := client.handleActions(); err != nil {
		log.Fatal(err)
	}
}

type client struct {
	conn     *websocket.Conn
	scanner  *bufio.Scanner
	board    domain.Board
	cellType domain.Cell
}

func newClient(conn *websocket.Conn) *client {
	return &client{
		conn:    conn,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (c *client) handleActions() error {
	for {
		msg := new(domain.Message)
		if err := c.conn.ReadJSON(msg); err != nil {
			return errors.WithMessage(err, "read json msg")
		}
		switch msg.Type {
		case domain.StartGame:
			if err := c.handleStartGameAction(msg); err != nil {
				return errors.WithMessage(err, "handle start game action")
			}
		case domain.RequestMove:
			if err := c.handleRequestMoveAction(); err != nil {
				return errors.WithMessage(err, "handle request move action")
			}
		case domain.PlayerMove:
			isGameFinished, err := c.handlePlayerMoveAction(msg)
			if err != nil {
				return errors.WithMessage(err, "handle player move action")
			}
			if isGameFinished {
				return nil
			}
		}
	}
}

func (c *client) handleStartGameAction(msg *domain.Message) error {
	v, err := utils.UnmarshalJson[domain.StartGamePayload](msg.Payload)
	if err != nil {
		return errors.WithMessage(err, "unmarshal json to 'StartGamePayload' type")
	}
	c.cellType = v.CellType
	c.board = v.Board
	c.printBoard()
	return nil
}

func (c *client) handleRequestMoveAction() error {
	var (
		pos byte
		err error
	)
	for {
		fmt.Printf("Твой ход: ")
		pos, err = c.selectCell()
		if err == nil {
			break
		}
		fmt.Print("\033[F\033[K")
	}
	err = c.conn.WriteJSON(domain.Message{
		Type: domain.PlayerMove,
		Payload: domain.PlayerMovePayload{
			CellType: c.cellType,
			Position: pos,
		},
	})
	if err != nil {
		return errors.WithMessage(err, "write json msg")
	}
	return nil
}

func (c *client) handlePlayerMoveAction(msg *domain.Message) (isGameFinished bool, err error) {
	v, err := utils.UnmarshalJson[domain.PlayerMovePayload](msg.Payload)
	if err != nil {
		return false, errors.WithMessage(err, "unmarshal json to 'PlayerMovePayload' type")
	}
	c.board[v.Position] = v.CellType
	c.printBoard()
	if v.GameResult != nil {
		fmt.Println(*v.GameResult)
		return true, nil
	}
	if v.IsMoveRequested {
		if err := c.handleRequestMoveAction(); err != nil {
			return false, errors.WithMessage(err, "handle request move action")
		}
	}
	return false, nil
}

func (c *client) selectCell() (byte, error) {
	if ok := c.scanner.Scan(); !ok {
		return 0, c.scanner.Err()
	}
	pos, err := strconv.ParseUint(c.scanner.Text(), 10, 8)
	if err != nil {
		return 0, err
	}
	return byte(pos - 1), nil
}

func (c *client) printBoard() {
	fmt.Printf("\033[H\033[J")
	for i, cell := range c.board {
		if (i+1)%3 == 0 {
			fmt.Printf("%c ", cell)
			if i < 6 {
				fmt.Printf("\n——|———|——\n")
			}
		} else {
			fmt.Printf("%c | ", cell)
		}
	}
	fmt.Println()
}
