package domain

type Player struct {
	uuid      string
	gameUuid  string
	playerCli Client
	cell      Cell
	ch        chan Move
}

func NewPlayer(gameUuid string, cli Client, cellType Cell, ch chan Move) Player {
	return Player{
		uuid:      cli.Uuid(),
		gameUuid:  gameUuid,
		playerCli: cli,
		cell:      cellType,
		ch:        ch,
	}
}

func (p Player) Uuid() string {
	return p.uuid
}

func (p Player) GameUuid() string {
	return p.gameUuid
}

func (p Player) SendMessage(msg Message) error {
	return p.playerCli.WriteMessage(msg)
}

func (p Player) ReceiveMessage() (Message, error) {
	return p.playerCli.ReadMessage()
}

func (p Player) Cell() Cell {
	return p.cell
}

func (p Player) GetEnemyMove() <-chan Move {
	return p.ch
}

func (p Player) MakeMove(move Move) {
	p.ch <- move
}
