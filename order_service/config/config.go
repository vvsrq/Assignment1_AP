package config

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

type Config struct {
	DatabaseURL              string `envconfig:"DATABASE_URL"              required:"true"`
	GrpcPort                 string `envconfig:"GRPC_PORT"                 default:":50052"`
	LogLevel                 string `envconfig:"LOG_LEVEL"                 default:"info"`
	InventoryServiceGrpcAddr string `envconfig:"INVENTORY_SERVICE_GRPC_ADDR" required:"true"`
}

var (
	config Config
	once   sync.Once
)

func LoadConfig(logger *logrus.Logger) *Config {
	once.Do(func() {
		err := godotenv.Load()
		if err != nil && !os.IsNotExist(err) {
			logger.Warnf("Error loading .env file (but continuing): %v", err)
		} else if err == nil {
			logger.Info("Loaded configuration from .env file")
		}

		err = envconfig.Process("", &config)
		if err != nil {
			logger.Fatalf("Failed to process configuration from environment variables: %v", err)
		}

		logger.Infof("Configuration loaded: GRPC Port=%s, LogLevel=%s, InventoryServiceGrpcAddr=%s",
			config.GrpcPort, config.LogLevel, config.InventoryServiceGrpcAddr)
		if config.DatabaseURL != "" {
			logger.Info("Configuration loaded: DatabaseURL is set")
		} else {
			logger.Fatal("Configuration error: DATABASE_URL is not set")
		}
		if config.InventoryServiceGrpcAddr == "" {
			logger.Fatal("Configuration error: INVENTORY_SERVICE_GRPC_ADDR is not set")
		}

	})
	return &config
}

func GetConfig() *Config {
	if config.GrpcPort == "" || config.DatabaseURL == "" || config.InventoryServiceGrpcAddr == "" {
		log.Fatal("Configuration not loaded. Call LoadConfig first.")
	}
	return &config
}
