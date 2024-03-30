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

type SyncUseCase interface {
	Sync(ctx context.Context)
	Addresses() map[string]string
	DefineMasterServer(ctx context.Context, req *DefineMasterRequest) (DefineMasterResponse, error)
	DefineServerRole(ctx context.Context, masterToIgnore string)
	CheckMasterHealth(ctx context.Context) error
	Chan() <-chan ServerInfo
}

type SyncRepository interface {
	Sync(ctx context.Context, addr string, states map[string]*GameState) error
	DefineMaster(ctx context.Context, req DefineMasterRequest, addr string) (*DefineMasterResponse, error)
	HealthCheck(ctx context.Context, addr string) error
}
