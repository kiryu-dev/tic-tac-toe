package webapi

import (
	"bytes"
	"context"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/kiryu-dev/tic-tac-toe/internal/domain"
	"github.com/pkg/errors"
)

const (
	clientTimeout       = 5 * time.Second
	syncStatesEndpoint  = "/sync"
	healthCheckEndpoint = "/health"
)

type repository struct {
	cli *http.Client
}

func New() repository {
	return repository{
		cli: &http.Client{Timeout: clientTimeout},
	}
}

func (r repository) Sync(ctx context.Context, addr string, states map[string]*domain.GameState) error {
	body, err := jsoniter.Marshal(states)
	if err != nil {
		return errors.WithMessage(err, "marshal json body")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+syncStatesEndpoint, bytes.NewReader(body))
	if err != nil {
		return errors.WithMessage(err, "new post request")
	}
	resp, err := r.cli.Do(req)
	if err != nil {
		return errors.WithMessagef(err, "call http endpoint '%s'", syncStatesEndpoint)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected response status '%s'", resp.Status)
	}
	return nil
}

func (r repository) HealthCheck(ctx context.Context, addr string) (*domain.HealthCheckResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, addr+healthCheckEndpoint, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "new get request")
	}
	resp, err := r.cli.Do(request)
	if err != nil {
		return nil, errors.WithMessagef(err, "call http endpoint '%s'", healthCheckEndpoint)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected response status '%s'", resp.Status)
	}
	result := new(domain.HealthCheckResponse)
	if err := jsoniter.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, errors.WithMessage(err, "decode json response body")
	}
	_ = resp.Body.Close()
	return result, nil
}
