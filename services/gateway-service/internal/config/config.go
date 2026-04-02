package config

import (
	"time"
)

type Config struct {
	App struct {
		Name        string
		Version     string
		Environment string
	}
	Server struct {
		Host         string
		Port         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}
	GRPC struct {
		UserServiceAddr  string
		BookServiceAddr  string
		OrderServiceAddr string
	}
	JWT struct {
		Secret    []byte
		Algorithm string
		ExpMins   time.Duration
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	RateLimit struct {
		MaxRequests   int32
		WindowSeconds int32
	}
	CORSMiddleware struct {
		MaxAge int
	}
}

type ConfigLoader interface {
	GetConfig() *Config
}
