package config

import (
	libHttp "github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/libs/go/config"
)

type Config struct {
	HTTP libHttp.Config	
}

func Load() (*Config, error) {
	cfg := Config{}

	cfg.HTTP.Port = config.GetEnv("HTTP_PORT", 8881)
	
	return &cfg, nil
}