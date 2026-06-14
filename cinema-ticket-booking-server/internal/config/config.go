package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	MongoURI    string
	MongoDB     string
	RedisAddr   string
	RabbitMQURL string
	CORSOrigins []string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Port:        getEnv("PORT", "8080"),
		MongoURI:    getEnv("MONGO_URI", ""),
		MongoDB:     getEnv("MONGO_DB", "cinema"),
		RedisAddr:   getEnv("REDIS_ADDR", "redis:6379"),
		RabbitMQURL: getEnv("RABBITMQ_URL", ""),
		CORSOrigins: parseList(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseList splits a comma-separated string into a trimmed slice.
func parseList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}