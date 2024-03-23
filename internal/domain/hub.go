package domain

import (
	"context"
)

type HubUseCase interface {
	Handle(ctx context.Context, client Client) error
}
