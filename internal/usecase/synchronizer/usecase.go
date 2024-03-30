package synchronizer

import (
	"context"
	"os"
	"time"

	"github.com/kiryu-dev/tic-tac-toe/internal/config"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type useCase struct {
	repo       domain.SyncRepository
	ch         chan map[string]*domain.GameState
	addrs      map[string]string
	masterName *atomic.String
	serverName string
	//mu         *sync.RWMutex
	logger  *zap.Logger
	ticker  *time.Ticker
	srvChan chan domain.ServerInfo
}

const (
	httpPrefix        = "http://"
	healthCheckPeriod = 5 * time.Second
)

func New(repo domain.SyncRepository, statesChan chan map[string]*domain.GameState, cfg []config.ServerConfig,
	logger *zap.Logger) *useCase {
	addrs := make(map[string]string)
	serverName := os.Getenv("SERVER_NAME")
	port := os.Getenv("SERVER_PORT")
	logger.Info("server name: " + serverName)
	for _, addr := range cfg {
		if addr.Host != serverName {
			fullAddr := httpPrefix + addr.Host + port
			addrs[addr.Host] = fullAddr
		}
	}
	logger.Info("defined servers", zap.Any("servers", addrs))
	logger.Info("define master server", zap.String("host", serverName))
	return &useCase{
		repo:       repo,
		ch:         statesChan,
		addrs:      addrs,
		serverName: serverName,
		masterName: atomic.NewString(serverName),
		//mu:         &sync.RWMutex{},
		logger:  logger,
		ticker:  time.NewTicker(healthCheckPeriod),
		srvChan: make(chan domain.ServerInfo),
	}
}

func (u *useCase) Sync(ctx context.Context) {
	for {
		select {
		case v := <-u.ch:
			u.logger.Info("starting sync games states...")
			for addr := range u.addrs {
				err := u.repo.Sync(ctx, addr, v)
				if err != nil {
					u.logger.Warn(err.Error())
				}
			}
		}
	}
}

func (u *useCase) DefineMasterServer(_ context.Context, req *domain.DefineMasterRequest) (domain.DefineMasterResponse, error) {
	u.logger.Info("define master server", zap.Any("req", req))
	if req.MasterToIgnore != "" && req.MasterToIgnore == u.masterName.Load() {
		u.masterName.Store(u.serverName)
	}

	if req.InitiatorMasterServerName == u.masterName.Load() {
		return domain.DefineMasterResponse{
			MasterServerName: u.masterName.Load(),
		}, nil
	}

	if _, ok := u.addrs[req.InitiatorMasterServerName]; !ok {
		u.logger.Warn("undefined server", zap.String("host", req.InitiatorMasterServerName))
		return domain.DefineMasterResponse{}, errors.New("undefined server")
	}

	u.compareWithCurrentMaster(req.InitiatorMasterServerName)

	return domain.DefineMasterResponse{
		MasterServerName: u.masterName.Load(),
	}, nil
}

func (u *useCase) DefineServerRole(ctx context.Context, masterToIgnore string) {
	for host, addr := range u.addrs {
		if host == masterToIgnore {
			continue
		}
		resp, err := u.repo.DefineMaster(ctx, domain.DefineMasterRequest{
			InitiatorMasterServerName: u.masterName.Load(),
			MasterToIgnore:            masterToIgnore,
		}, addr)
		if err != nil {
			u.logger.Warn(err.Error())
			continue
		}
		u.compareWithCurrentMaster(resp.MasterServerName)
	}

	serverRole := domain.ReserveServer
	masterName := u.masterName.Load()
	if masterName == u.serverName {
		serverRole = domain.MasterServer
	}

	u.srvChan <- domain.ServerInfo{
		ServerRole:       serverRole,
		MasterServerName: masterName,
	}
}

func (u *useCase) CheckMasterHealth(ctx context.Context) error {
	for range u.ticker.C {
		masterName := u.masterName.Load()
		if masterName == u.serverName {
			break
		}
		u.logger.Info("starting to check master server's health", zap.String("host", masterName))

		masterAddr, ok := u.addrs[masterName]
		if !ok {
			return errors.Errorf("undefined master server '%s'", u.masterName)
		}

		err := u.repo.HealthCheck(ctx, masterAddr)
		if err != nil {
			u.logger.Warn(err.Error())
			if u.masterName.Load() == masterName {
				u.masterName.Store(u.serverName)
			}
			go u.DefineServerRole(ctx, masterName)
			break
		}
	}
	return nil
}

func (u *useCase) compareWithCurrentMaster(newMasterServerName string) {
	masterName := u.masterName.Load()
	if masterName == "" || masterName > newMasterServerName {
		u.masterName.Store(newMasterServerName)
		u.logger.Info("new master server", zap.String("host", u.masterName.Load()))
	}
}

func (u *useCase) Addresses() map[string]string {
	return u.addrs
}

func (u *useCase) Chan() <-chan domain.ServerInfo {
	return u.srvChan
}
