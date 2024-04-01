package domain

import (
	"context"
)

type ServerRole string

const (
	MasterServer  = ServerRole("master")
	ReserveServer = ServerRole("reserve")
)

type ServerInfo struct {
	ServerRole       ServerRole
	MasterServerName string
}

type DefineMasterRequest struct {
	InitiatorMasterServerName string
	MasterToIgnore            string
}

type DefineMasterResponse struct {
	MasterServerName string
}

type HealthCheckResponse struct {
	Role ServerRole
}

type SyncUseCase interface {
	Sync(ctx context.Context, statesChan <-chan map[string]*GameState)
	DefineMasterServer(ctx context.Context)
	CheckMasterHealth(ctx context.Context) error
	Chan() <-chan ServerInfo
}

type SyncRepository interface {
	Sync(ctx context.Context, addr string, states map[string]*GameState) error
	HealthCheck(ctx context.Context, addr string) (*HealthCheckResponse, error)
}
