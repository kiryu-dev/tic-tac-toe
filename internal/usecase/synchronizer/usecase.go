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
	addrs      map[string]string
	masterName *atomic.String
	serverName string
	logger     *zap.Logger
	ticker     *time.Ticker
	srvChan    chan domain.ServerInfo
}

const (
	httpPrefix        = "http://"
	healthCheckPeriod = 2 * time.Second
)

func New(repo domain.SyncRepository, cfg []config.ServerConfig,
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
	return &useCase{
		repo:       repo,
		addrs:      addrs,
		serverName: serverName,
		masterName: atomic.NewString(""),
		logger:     logger,
		ticker:     time.NewTicker(healthCheckPeriod),
		srvChan:    make(chan domain.ServerInfo),
	}
}

func (u *useCase) Sync(ctx context.Context, statesChan <-chan map[string]*domain.GameState) {
	for {
		select {
		case v := <-statesChan:
			if u.serverName != u.masterName.Load() {
				continue
			}
			u.logger.Info("starting sync games states...")
			for _, addr := range u.addrs {
				err := u.repo.Sync(ctx, addr, v)
				if err != nil {
					u.logger.Warn(err.Error())
				}
			}
		}
	}
}

func (u *useCase) DefineMasterServer(ctx context.Context) {
	master := u.serverName
	for host, addr := range u.addrs {
		res, err := u.repo.HealthCheck(ctx, addr)
		if err != nil {
			u.logger.Warn(err.Error())
			continue
		}
		if res.Role == domain.MasterServer {
			master = host
			break
		}
		master = u.compareMasters(master, host)
	}

	serverRole := domain.ReserveServer
	u.masterName.Store(master)
	if master == u.serverName {
		serverRole = domain.MasterServer
	}

	u.srvChan <- domain.ServerInfo{
		ServerRole:       serverRole,
		MasterServerName: master,
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

		_, err := u.repo.HealthCheck(ctx, masterAddr)
		if err != nil {
			u.logger.Warn(err.Error())
			u.masterName.Store(u.serverName)
			go u.DefineMasterServer(ctx)
			break
		}
	}
	return nil
}

func (u *useCase) compareMasters(lhs string, rhs string) string {
	u.logger.Info("compare masters", zap.String("lhs", lhs), zap.String("rhs", rhs))
	if lhs > rhs {
		return rhs
	}
	return lhs
}

func (u *useCase) ServerInfoChan() <-chan domain.ServerInfo {
	return u.srvChan
}
