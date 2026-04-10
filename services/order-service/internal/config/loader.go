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
	cfg.App.Name = getEnvStrOrDefault("APP_NAME", "lms-order-service")
	cfg.App.Environment = getEnvStrOrDefault("APP_ENV", "development")
	cfg.App.Version = getEnvStrOrDefault("APP_VERSION", "1.0.0")

	// Server settings
	cfg.Server.Host = getEnvStrOrDefault("SERVER_HOST", "localhost")
	cfg.Server.Port = getEnvStrOrDefault("SERVER_PORT", "40043")

	// Database settings
	cfg.Database.DBHost = getEnvStrOrDefault("DB_HOST", "localhost")
	cfg.Database.DBPort = getEnvStrOrDefault("DB_PORT", "5432")
	cfg.Database.DBUser = getEnvStrOrDefault("DB_USER", "postgres")
	cfg.Database.DBPwd = getEnvStrOrDefault("DB_PASSWORD", "postgres")
	cfg.Database.DBName = getEnvStrOrDefault("DB_NAME", "lms_order_db")
	cfg.Database.DBSSLMode = getEnvStrOrDefault("DB_SSL_MODE", "disable")
	cfg.Database.DBMaxOpenConns = getEnvIntOrDefault("DB_MAX_OPEN_CONNS", 25)
	cfg.Database.DBMaxIdleConns = getEnvIntOrDefault("DB_MAX_IDLE_CONNS", 10)
	cfg.Database.DBConnMaxLifetime = getEnvDurationOrDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute)

	// Dependent services
	cfg.Services.UserServiceAddr = getEnvStrOrDefault("USER_SERVICE_ADDR", "localhost:40041")
	cfg.Services.BookServiceAddr = getEnvStrOrDefault("BOOK_SERVICE_ADDR", "localhost:40042")

	// RabbitMQ settings
	cfg.RabbitMQ.URL = getEnvStrOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	cfg.RabbitMQ.Exchange = getEnvStrOrDefault("RABBITMQ_EXCHANGE", "order-events")

	// SQS settings
	cfg.SQS.QueueURL = getEnvStrOrDefault("SQS_QUEUE_URL", "")
	cfg.SQS.Region = getEnvStrOrDefault("AWS_REGION", "ap-southeast-2")
	cfg.SQS.AccessKeyID = getEnvStrOrDefault("AWS_ACCESS_KEY_ID", "")
	cfg.SQS.SecretAccessKey = getEnvStrOrDefault("AWS_SECRET_ACCESS_KEY", "")

	// Broker type: "rabbitmq" (default for dev/test) or "sqs" (prod)
	cfg.BrokerType = getEnvStrOrDefault("BROKER_TYPE", "rabbitmq")

	return &envLoader{cfg: cfg}, nil
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
