package ws

import (
	"fmt"
	"net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"go.uber.org/zap"
)

func (s *server) serveWs(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("new connection", zap.String("master host", s.masterHost), zap.Any("role", s.role))
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}
	clientUuid := strings.TrimSpace(r.Header.Get(domain.ClientUuidHeader))
	if clientUuid == "" {
		s.logger.Warn(fmt.Sprintf("empty '%s' header", domain.ClientUuidHeader))
		return
	}
	client := newClient(conn, clientUuid)
	defer client.Close()
	switch s.role {
	case domain.ReserveServer:
		s.logger.Info("request client to switch server", zap.String("master host", s.masterHost))
		err := client.WriteMessage(domain.Message{
			Type:    domain.SwitchServer,
			Payload: domain.SwitchServerPayload{MasterServer: s.masterHost},
		})
		if err != nil {
			s.logger.Error(err.Error())
		}
	case domain.MasterServer:
		if err := s.hub.Handle(r.Context(), client); err != nil {
			s.logger.Error(err.Error())
		}
	default:
		s.logger.Warn("the client connected before the server role was determined")
	}
}

func (s *server) healthCheck(w http.ResponseWriter, _ *http.Request) {
	s.logger.Info("health checking...")
	resp := domain.HealthCheckResponse{
		MasterServer: s.masterHost,
		Role:         s.role,
	}
	if err := jsoniter.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Warn(err.Error())
	}
}

func (s *server) applyStates(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("sync states")
	req := make(map[string]*domain.GameState)
	if err := jsoniter.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.logger.Warn(err.Error())
		return
	}
	s.hub.ApplyStates(r.Context(), req)
}
