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
	cfg.App.Name = getEnvStrOrDefault("APP_NAME", "lms-user-service")
	cfg.App.Environment = getEnvStrOrDefault("APP_ENV", "development")
	cfg.App.Version = getEnvStrOrDefault("APP_VERSION", "1.0.0")

	// Server settings
	cfg.Server.Host = getEnvStrOrDefault("SERVER_HOST", "localhost")
	cfg.Server.Port = getEnvStrOrDefault("SERVER_PORT", "40041")

	// Database settings
	cfg.Database.DBHost = getEnvStrOrDefault("DB_HOST", "localhost")
	cfg.Database.DBPort = getEnvStrOrDefault("DB_PORT", "5432")
	cfg.Database.DBUser = getEnvStrOrDefault("DB_USER", "postgres")
	cfg.Database.DBPwd = getEnvStrOrDefault("DB_PASSWORD", "postgres")
	cfg.Database.DBName = getEnvStrOrDefault("DB_NAME", "lms_user_db")
	cfg.Database.DBSSLMode = getEnvStrOrDefault("DB_SSL_MODE", "disable")
	cfg.Database.DBMaxOpenConns = getEnvIntOrDefault("DB_MAX_OPEN_CONNS", 100)
	cfg.Database.DBMaxIdleConns = getEnvIntOrDefault("DB_MAX_IDLE_CONNS", 100)
	cfg.Database.DBConnMaxLifetime = getEnvDurationOrDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute)

	// JWT settings
	cfg.JWT.JWTSecret = []byte(getEnvStrOrDefault("JWT_SECRET", "lms-secret-key"))
	cfg.JWT.JWTAlgorithm = getEnvStrOrDefault("JWT_ALGORITHM", "HS256")
	cfg.JWT.JWTExpMins = getEnvDurationOrDefault("JWT_EXP_MINS", 60*time.Minute)

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
