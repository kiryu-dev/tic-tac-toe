package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kiryu-dev/tic-tac-toe/internal/config"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/kiryu-dev/tic-tac-toe/pkg/utils"
	"github.com/pkg/errors"
)

const connectTryPeriod = 3 * time.Second

var (
	clientUuid = uuid.NewString()
	portByHost = make(map[string]string)
)

func main() {
	cfgPath := flag.String("config", "./conf/config.yml", "path to config")
	flag.Parse()
	cfg, err := config.New(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, serverCfg := range cfg.Servers {
		portByHost[serverCfg.Host] = strconv.Itoa(serverCfg.Port)
	}
	ticker := time.NewTicker(connectTryPeriod)
	defer ticker.Stop()
	for {
		for host, port := range portByHost {
			err := сonnectToServer(port, ticker)
			if err == nil {
				return
			}
			log.Printf("Не удалось подключиться к серверу '%s': %v", host, err)
		}
	}
}

func сonnectToServer(port string, ticker *time.Ticker) error {
	for range ticker.C {
		serverAddress := net.JoinHostPort("localhost", port)
		u := url.URL{Scheme: "ws", Host: serverAddress, Path: "/game"}
		log.Printf("try to connect to server '%s'...\n", serverAddress)
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), map[string][]string{
			domain.ClientUuidHeader: {clientUuid},
		})
		if err != nil {
			return errors.WithMessage(err, "websocket dial")
		}
		client := newClient(conn)
		result, err := client.handleActions()
		_ = conn.Close()
		if err != nil {
			return errors.WithMessage(err, "handle actions")
		}
		if !result.shouldSwitchToNewMaster { // буквально означает, что игра закончена
			return nil
		}
		var ok bool
		port, ok = portByHost[result.newMasterServer]
		if !ok {
			return errors.New("undefined master server")
		}
		log.Printf("переключаемся на сервер '%s'\n", result.newMasterServer)
	}
	return nil
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

type handleActionsResult struct {
	shouldSwitchToNewMaster bool
	newMasterServer         string
}

func (c *client) handleActions() (handleActionsResult, error) {
	for {
		msg := new(domain.Message)
		if err := c.conn.ReadJSON(msg); err != nil {
			return handleActionsResult{}, errors.WithMessage(err, "read json msg")
		}
		switch msg.Type {
		case domain.StartGame:
			if err := c.handleStartGameAction(msg); err != nil {
				return handleActionsResult{}, errors.WithMessage(err, "handle start game action")
			}
		case domain.RequestMove:
			if err := c.handleRequestMoveAction(); err != nil {
				return handleActionsResult{}, errors.WithMessage(err, "handle request move action")
			}
		case domain.PlayerMove:
			isGameFinished, err := c.handlePlayerMoveAction(msg)
			if err != nil {
				return handleActionsResult{}, errors.WithMessage(err, "handle player move action")
			}
			if isGameFinished {
				return handleActionsResult{}, nil
			}
		case domain.Walkover:
			v, err := utils.UnmarshalJson[domain.WalkoverPayload](msg.Payload)
			if err != nil {
				return handleActionsResult{}, errors.WithMessage(err, "unmarshal json to 'WalkoverPayload' type")
			}
			fmt.Println(v.GameResult)
			return handleActionsResult{}, nil
		case domain.SwitchServer:
			v, err := utils.UnmarshalJson[domain.SwitchServerPayload](msg.Payload)
			if err != nil {
				return handleActionsResult{}, errors.WithMessage(err, "unmarshal json to 'SwitchServerPayload' type")
			}
			return handleActionsResult{
				shouldSwitchToNewMaster: true,
				newMasterServer:         v.MasterServer,
			}, nil
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
