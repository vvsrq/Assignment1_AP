package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	Port                string
	LogLevel            string
	InventoryServiceURL string
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback != "" {
		log.Printf("Warning: Environment variable %s not set, using default value: %s\n", key, fallback)
	} else {
		log.Printf("Warning: Environment variable %s not set and no default value provided\n", key)
	}
	return fallback
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Warning: .env file not found, using environment variables or defaults.")
		} else {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	}

	return &Config{
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		Port:                getEnv("ORDER_SERVICE_PORT", ":8082"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		InventoryServiceURL: getEnv("INVENTORY_SERVICE_URL", "http://localhost:8081"),
	}
}
