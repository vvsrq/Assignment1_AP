package main

import (
	"context" // Import context
	"database/sql"
	"fmt"
	"net"
	"order_service/config"
	"order_service/internal/clients"
	grpcHandler "order_service/internal/delivery/grpc"
	"order_service/internal/repository"
	"order_service/internal/usecase"
	orderpb "order_service/proto"
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
	logger.Infof("Starting Order Service (gRPC)...")

	database, err := connectDB(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer func() {
		logger.Info("Closing database connection...")
		if err := database.Close(); err != nil {
			logger.Errorf("Error closing database: %v", err)
		}
	}()

	invClient, err := clients.NewInventoryGRPCClient(cfg.InventoryServiceGrpcAddr, logger, 5*time.Second)
	if err != nil {
		logger.Fatalf("FATAL: Failed to create Inventory gRPC client: %v", err)
	}
	// TODO: Add defer invClient.Close() - requires Close() method in interface/implementation

	orderRepo := repository.NewPostgresOrderRepository(database, logger)
	logger.Info("Repositories initialized.")

	orderUseCase := usecase.NewOrderUseCase(orderRepo, invClient, logger)
	logger.Info("Use cases initialized.")

	orderGrpcHandler := grpcHandler.NewOrderHandler(orderUseCase, logger)
	logger.Info("gRPC Handler initialized.")

	lis, err := net.Listen("tcp", cfg.GrpcPort)
	if err != nil {
		logger.Fatalf("Failed to listen on port %s: %v", cfg.GrpcPort, err)
	}
	logger.Infof("gRPC server listening on %s", cfg.GrpcPort)

	grpcServer := grpc.NewServer()

	orderpb.RegisterOrderServiceServer(grpcServer, orderGrpcHandler)

	reflection.Register(grpcServer)
	logger.Info("gRPC reflection service registered")

	serverErrChan := make(chan error, 1)
	go func() {
		logger.Info("Starting gRPC server...")
		err := grpcServer.Serve(lis)
		if err != nil && err != grpc.ErrServerStopped {
			logger.Errorf("gRPC server failed to serve: %v", err)
			serverErrChan <- err
		} else {
			logger.Info("gRPC server stopped serving gracefully.")
			close(serverErrChan)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	logger.Info("Signal listener started.")

	select {
	case sig := <-quit:
		logger.Warnf("Shutdown signal received: %v", sig)
	case err := <-serverErrChan:
		if err != nil {
			logger.Errorf("gRPC server failed unexpectedly: %v", err)
		}
	}

	logger.Info("Attempting graceful shutdown of gRPC server...")
	grpcServer.GracefulStop()
	logger.Info("gRPC server gracefully stopped.")

	if clientWithCloser, ok := invClient.(interface{ Close() error }); ok {
		logger.Info("Closing Inventory gRPC client connection...")
		if err := clientWithCloser.Close(); err != nil {
			logger.Errorf("Error closing inventory client: %v", err)
		}
	}

	logger.Info("Order Service shut down gracefully.")

}

func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetOutput(os.Stdout)
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	return logger
}

func connectDB(dataSourceName string, logger *logrus.Logger) (*sql.DB, error) {
	logger.Info("Connecting to database...")
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	logger.Info("Database connection established successfully.")
	return db, nil
}
