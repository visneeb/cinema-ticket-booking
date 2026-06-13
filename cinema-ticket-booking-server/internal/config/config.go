package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDB        string
	RedisAddr      string
	RabbitMQURL    string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Port:           getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGO_URI", ""),
		MongoDB:        getEnv("MONGO_DB", "cinema"),
		RedisAddr:      getEnv("REDIS_ADDR", "redis:6379"),
		RabbitMQURL:    getEnv("RABBITMQ_URL", "amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}