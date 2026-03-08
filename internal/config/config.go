package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	Kafka    KafkaConfig
	API      APIConfig
	Worker   WorkerConfig
	Log      LogConfig
}

type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type KafkaConfig struct {
	Brokers               []string
	TopicNotifications    string
	TopicNotificationsDLQ string
	ConsumerGroup         string
}

type APIConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type WorkerConfig struct {
	Consumers     int
	RetryAttempts int
	RetryDelay1   time.Duration
	RetryDelay2   time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	return &Config{
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			Name:            getEnv("DB_NAME", "notification_db"),
			User:            getEnv("DB_USER", "notification_user"),
			Password:        getEnv("DB_PASSWORD", "notification_password"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Kafka: KafkaConfig{
			Brokers:               getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}, ","),
			TopicNotifications:    getEnv("KAFKA_TOPIC_NOTIFICATIONS", "notifications"),
			TopicNotificationsDLQ: getEnv("KAFKA_TOPIC_NOTIFICATIONS_DLQ", "notifications-dlq"),
			ConsumerGroup:         getEnv("KAFKA_CONSUMER_GROUP", "notification-workers"),
		},
		API: APIConfig{
			Port:         getEnv("API_PORT", "8080"),
			ReadTimeout:  getEnvAsDuration("API_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvAsDuration("API_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvAsDuration("API_IDLE_TIMEOUT", 60*time.Second),
		},
		Worker: WorkerConfig{
			Consumers:     getEnvAsInt("WORKER_CONSUMERS", 3),
			RetryAttempts: getEnvAsInt("WORKER_RETRY_ATTEMPTS", 3),
			RetryDelay1:   getEnvAsDuration("WORKER_RETRY_DELAY_1", 30*time.Second),
			RetryDelay2:   getEnvAsDuration("WORKER_RETRY_DELAY_2", 2*time.Minute),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsDuration(name string, defaultVal time.Duration) time.Duration {
	valueStr := getEnv(name, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsSlice(name string, defaultVal []string, sep string) []string {
	valueStr := getEnv(name, "")
	if valueStr == "" {
		return defaultVal
	}
	return splitString(valueStr, sep)
}

func splitString(s string, sep string) []string {
	var result []string
	for _, token := range split(s, sep) {
		result = append(result, token)
	}
	return result
}

func split(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	return []string{s}
}
