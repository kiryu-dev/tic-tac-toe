package config

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var ErrNotEnoughServers = errors.New("there are not enough specified servers")

const minServerCount = 2

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type config struct {
	Servers []ServerConfig `yaml:"outer_servers"`
}

func New(cfgPath string) (config, error) {
	file, err := os.Open(cfgPath)
	if err != nil {
		return config{}, err
	}
	defer func() {
		_ = file.Close()
	}()
	cfg := config{}
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return config{}, err
	}
	if len(cfg.Servers) < minServerCount {
		return config{}, ErrNotEnoughServers
	}
	return cfg, nil
}
