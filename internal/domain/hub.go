package domain

import (
	"context"
)

type HubUseCase interface {
	Handle(ctx context.Context, client Client) error
	GamesStates() <-chan map[string]*GameState
	ApplyStates(ctx context.Context, states map[string]*GameState)
}
