package config

import (
	"time"
)

type Config struct {
	App struct {
		Name        string
		Environment string
		Version     string
	}
	Server struct {
		Host string
		Port string
	}
	Database struct {
		DBHost            string
		DBPort            string
		DBUser            string
		DBPwd             string
		DBName            string
		DBSSLMode         string
		DBMaxOpenConns    int
		DBMaxIdleConns    int
		DBConnMaxLifetime time.Duration
	}
	JWT struct {
		JWTSecret    []byte
		JWTAlgorithm string
		JWTExpMins   time.Duration
	}
}

type ConfigLoader interface {
	GetConfig() *Config
}
