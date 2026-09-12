package config

import (
	"github.com/niaga-labs/niaga-labs-pet-lib-common/config"
)

// ServiceConfig holds all configuration for the runner service.
type ServiceConfig struct {
	Port        string
	AppEnv      string
	DBConfig    config.DatabaseConfig
	JWTConfig   config.JWTConfig
	KafkaConfig config.KafkaConfig
}

// Load reads configuration from environment variables and returns a ServiceConfig.
func Load() (*ServiceConfig, error) {
	v, err := config.Load("RUNNER")
	if err != nil {
		return nil, err
	}

	return &ServiceConfig{
		Port:        config.GetServicePort(v, "SERVICE_PORT"),
		AppEnv:      config.GetAppEnv(v),
		DBConfig:    config.LoadDatabaseConfig(v, "DB_NAME"),
		JWTConfig:   config.LoadJWTConfig(v),
		KafkaConfig: config.LoadKafkaConfig(v),
	}, nil
}
