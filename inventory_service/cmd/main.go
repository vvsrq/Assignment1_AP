package main

import (
	"context"
	"database/sql"
	"fmt"
	"inventory_service/config"
	grpcHandler "inventory_service/internal/delivery/grpc"
	"inventory_service/internal/repository"
	"inventory_service/internal/usecase"
	inventorypb "inventory_service/proto"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	logger := setupLogger("info")
	cfg := config.LoadConfig(logger)
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Warnf("Invalid log level '%s', using default 'info'. Error: %v", cfg.LogLevel, err)
	} else {
		logger.SetLevel(logLevel)
	}
	logger.Infof("Starting Inventory Service (gRPC)...")

	database, err := connectDB(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Errorf("Error closing database connection: %v", err)
		} else {
			logger.Info("Database connection closed.")
		}
	}()
	logger.Info("Database connection established.")

	categoryRepo := repository.NewPostgresCategoryRepository(database, logger)
	productRepo := repository.NewPostgresProductRepository(database, logger)
	logger.Info("Repositories initialized.")

	categoryUseCase := usecase.NewCategoryUseCase(categoryRepo, logger)
	productUseCase := usecase.NewProductUseCase(productRepo, categoryRepo, logger)
	logger.Info("Use cases initialized.")

	inventoryGrpcHandler := grpcHandler.NewInventoryHandler(productUseCase, categoryUseCase, logger)
	logger.Info("gRPC Handler initialized.")

	lis, err := net.Listen("tcp", cfg.GrpcPort)
	if err != nil {
		logger.Fatalf("Failed to listen on port %s: %v", cfg.GrpcPort, err)
	}
	logger.Infof("gRPC server listening on %s", cfg.GrpcPort)

	grpcServer := grpc.NewServer()

	inventorypb.RegisterInventoryServiceServer(grpcServer, inventoryGrpcHandler)

	reflection.Register(grpcServer)
	logger.Info("gRPC reflection service registered")

	go func() {
		logger.Info("Starting gRPC server...")
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			logger.Fatalf("Failed to serve gRPC: %v", err)
		}
		logger.Info("gRPC server stopped serving.")
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	logger.Info("Signal listener started.")

	<-quit // Block until signal
	logger.Warn("Shutdown signal received...")

	logger.Info("Attempting graceful shutdown of gRPC server...")
	grpcServer.GracefulStop()
	logger.Info("gRPC server gracefully stopped.")

	logger.Info("Inventory Service shut down gracefully.")

}

func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetOutput(os.Stdout)
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logger.Warnf("Invalid log level '%s', using default 'info'. Error: %v", level, err)
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	return logger
}

func connectDB(dataSourceName string, logger *logrus.Logger) (*sql.DB, error) {
	logger.Info("Connecting to database...")
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Info("Database connection established successfully.")
	return db, nil
}
