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
	JwtSecret                string `envconfig:"JWT_SECRET"                  required:"true"`
	GatewayPort              string `envconfig:"API_GATEWAY_PORT"            default:":8080"`
	LogLevel                 string `envconfig:"LOG_LEVEL"                   default:"info"`
	InventoryServiceGrpcAddr string `envconfig:"INVENTORY_SERVICE_GRPC_ADDR" required:"true"`
	OrderServiceGrpcAddr     string `envconfig:"ORDER_SERVICE_GRPC_ADDR"     required:"true"`
	UserServiceGrpcAddr      string `envconfig:"USER_SERVICE_GRPC_ADDR"      required:"true"`
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

		// Log loaded config
		logger.Infof("Configuration loaded: GatewayPort=%s, LogLevel=%s", config.GatewayPort, config.LogLevel)
		logger.Infof("UserServiceAddr=%s, InventoryServiceAddr=%s, OrderServiceAddr=%s",
			config.UserServiceGrpcAddr, config.InventoryServiceGrpcAddr, config.OrderServiceGrpcAddr)
		if config.JwtSecret == "" {
			logger.Fatal("Configuration error: JWT_SECRET is not set")
		}
		if config.InventoryServiceGrpcAddr == "" || config.OrderServiceGrpcAddr == "" || config.UserServiceGrpcAddr == "" {
			logger.Fatal("Configuration error: One or more gRPC service addresses are not set")
		}

	})
	return &config
}

func GetConfig() *Config {
	if config.GatewayPort == "" {
		log.Fatal("Configuration not loaded. Call LoadConfig first.")
	}
	return &config
}
