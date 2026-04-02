package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type envLoader struct {
	cfg *Config
}

func NewConfigLoader() (ConfigLoader, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	// App settings
	cfg.App.Name = getEnvStrOrDefault("APP_NAME", "lms-gateway-service")
	cfg.App.Version = getEnvStrOrDefault("APP_VERSION", "1.0.0")
	cfg.App.Environment = getEnvStrOrDefault("APP_ENV", "development")

	// Server settings
	cfg.Server.Host = getEnvStrOrDefault("SERVER_HOST", "localhost")
	cfg.Server.Port = getEnvStrOrDefault("SERVER_PORT", "8080")
	cfg.Server.ReadTimeout = getEnvDurationOrDefault("SERVER_READ_TIMEOUT", 5*time.Second)
	cfg.Server.WriteTimeout = getEnvDurationOrDefault("SERVER_WRITE_TIMEOUT", 10*time.Second)
	cfg.Server.IdleTimeout = getEnvDurationOrDefault("SERVER_IDLE_TIMEOUT", 120*time.Second)

	// gRPC service addresses
	cfg.GRPC.UserServiceAddr = getEnvStrOrDefault("GRPC_USER_SERVICE_ADDR", "localhost:40041")
	cfg.GRPC.BookServiceAddr = getEnvStrOrDefault("GRPC_BOOK_SERVICE_ADDR", "localhost:40042")
	cfg.GRPC.OrderServiceAddr = getEnvStrOrDefault("GRPC_ORDER_SERVICE_ADDR", "localhost:40043")

	// JWT settings
	cfg.JWT.Secret = []byte(getEnvStrOrDefault("JWT_SECRET", "lms-secret-key"))
	cfg.JWT.Algorithm = getEnvStrOrDefault("JWT_ALGORITHM", "HS256")
	cfg.JWT.ExpMins = getEnvDurationOrDefault("JWT_EXP_MINS", 60*time.Minute)

	// Redis settings
	cfg.Redis.Addr = getEnvStrOrDefault("REDIS_ADDR", "localhost:6379")
	cfg.Redis.Password = getEnvStrOrDefault("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvIntOrDefault("REDIS_DB", 0)

	// Rate limiting settings
	cfg.RateLimit.MaxRequests = int32(getEnvIntOrDefault("RATE_LIMIT_MAX_REQUESTS", 100))
	cfg.RateLimit.WindowSeconds = int32(getEnvIntOrDefault("RATE_LIMIT_WINDOW_SECONDS", 60))

	// CORS middleware settings
	cfg.CORSMiddleware.MaxAge = getEnvIntOrDefault("CORS_MAX_AGE", 3600)

	return &envLoader{
		cfg: cfg,
	}, nil
}

func (l *envLoader) GetConfig() *Config {
	return l.cfg
}

func getEnvStrOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func getEnvIntOrDefault(key string, fallback int) int {
	strVal := getEnvStrOrDefault(key, "")
	if strVal == "" {
		return fallback
	}

	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		return fallback
	}

	return intVal
}

func getEnvDurationOrDefault(key string, fallback time.Duration) time.Duration {
	strVal := getEnvStrOrDefault(key, "")
	if strVal == "" {
		return fallback
	}

	durVal, err := time.ParseDuration(strVal)
	if err != nil {
		return fallback
	}

	return durVal
}
