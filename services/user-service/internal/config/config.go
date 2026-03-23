package config

import (
	"os"
	"strconv"
	"time"
)

type SvcConfig struct {
	Environment string
	SvcName     string
	Version     string

	GRPCPort int

	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int

	JWTSecret    []byte
	JWTAlgorithm string
	JWTExpMins   time.Duration
}

func LoadFromEnv() *SvcConfig {
	return &SvcConfig{
		Environment: getEnvOrDefault("APP_ENV", "development"),
		SvcName:     getEnvOrDefault("SVC_NAME", "user-service"),
		Version:     getEnvOrDefault("VERSION", "1.0.0"),

		GRPCPort: getEnvAsInt("GRPC_PORT", 40041),

		DBHost:            getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:            getEnvOrDefault("DB_PORT", "5432"),
		DBUser:            getEnvOrDefault("DB_USER", "postgres"),
		DBPassword:        getEnvOrDefault("DB_PASSWORD", ""),
		DBName:            getEnvOrDefault("DB_NAME", "user_db"),
		DBSSLMode:         getEnvOrDefault("DB_SSL_MODE", "disable"),
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME_MINS", 5),

		JWTSecret:    []byte(getEnvOrDefault("JWT_SECRET", "gateway-secret-key")),
		JWTAlgorithm: getEnvOrDefault("JWT_ALGORITHM", "HS256"),
		JWTExpMins:   time.Duration(getEnvAsInt("JWT_EXP_MINS", 60)) * time.Minute,
	}
}

func getEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	val := getEnvOrDefault(key, "")
	if val == "" {
		return fallback
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return fallback
}
