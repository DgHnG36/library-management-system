package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type SvcConfig struct {
	Environment string
	SvcName     string
	Version     string
	Port        string

	JWTSecret    []byte
	JWTAlgorithm string
	JWTExpMins   time.Duration

	RedisAddr string
	RedisPwd  string
	RedisDB   int

	RateMaxRequests int32
	RateWindow      int32

	TargetMap map[string]string

	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
}

func LoadFromEnv() *SvcConfig {
	return &SvcConfig{
		Environment:      getEnvOrDefault("APP_ENV", "development"),
		SvcName:          getEnvOrDefault("SVC_NAME", "gateway-service"),
		Version:          getEnvOrDefault("VERSION", "1.0.0"),
		Port:             getEnvOrDefault("PORT", "8080"),
		JWTSecret:        []byte(getEnvOrDefault("JWT_SECRET", "gateway-secret-key")),
		JWTAlgorithm:     getEnvOrDefault("JWT_ALGORITHM", "HS256"),
		JWTExpMins:       getEnvAsDuration("JWT_EXP_MINS", 60),
		RedisAddr:        getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPwd:         getEnvOrDefault("REDIS_PWD", ""),
		RedisDB:          getEnvAsInt("REDIS_DB", 0),
		RateMaxRequests:  int32(getEnvAsInt("RATE_MAX_REQUESTS", 100)),
		RateWindow:       int32(getEnvAsInt("RATE_WINDOW_SECONDS", 60)),
		TargetMap:        parseTargetMap(getEnvOrDefault("TARGET_MAP", "/api/v1/users=http://localhost:40041,/api/v1/books=http://localhost:40042,/api/v1/orders=http://localhost:40043")),
		HTTPReadTimeout:  getEnvAsDuration("HTTP_READ_TIMEOUT", 30*time.Second),
		HTTPWriteTimeout: getEnvAsDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		HTTPIdleTimeout:  getEnvAsDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
	}
}

/* HELPER METHODS */
func getEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	strVal := getEnvOrDefault(key, "")
	if strVal == "" {
		return fallback
	}

	if val, err := strconv.Atoi(strVal); err == nil {
		return val
	}

	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	strVal := getEnvOrDefault(key, "")
	if strVal == "" {
		return fallback
	}

	if val, err := strconv.Atoi(strVal); err == nil {
		return time.Duration(val) * time.Minute
	}

	return fallback
}

func parseTargetMap(raw string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		result[kv[0]] = kv[1]
	}

	return result
}
