package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret           string
	InventoryServiceURL string
	OrderServiceURL     string
	GatewayPort         string
	LogLevel            string
}

func LoadConfig() *Config {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found or error loading it, using environment variables or defaults")
	}

	return &Config{
		JWTSecret:           getEnv("JWT_SECRET", "default_secret_change_me"), // Важно иметь дефолт, но небезопасный
		InventoryServiceURL: getEnv("INVENTORY_SERVICE_URL", "http://localhost:8081"),
		OrderServiceURL:     getEnv("ORDER_SERVICE_URL", "http://localhost:8082"),
		GatewayPort:         getEnv("API_GATEWAY_PORT", ":8080"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Warning: Environment variable %s not set, using default value: %s\n", key, fallback)
	return fallback
}
