package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user_service/internal/config"
	grpcHandler "user_service/internal/delivery/grpc"
	"user_service/internal/repository"
	"user_service/internal/usecase"
	userpb "user_service/proto"

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
		logger.Warnf("Invalid log level '%s' in config, using default 'info'. Error: %v", cfg.LogLevel, err)
	} else {
		logger.SetLevel(logLevel)
	}
	logger.Infof("Starting User Service...")

	db, err := connectDB(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Errorf("Error closing database connection: %v", err)
		} else {
			logger.Info("Database connection closed.")
		}
	}()

	userRepo := repository.NewPostgresUserRepository(db, logger)
	userUseCase := usecase.NewUserUseCase(userRepo, logger)
	userGrpcHandler := grpcHandler.NewUserHandler(userUseCase, logger)

	lis, err := net.Listen("tcp", cfg.GrpcPort)
	if err != nil {
		logger.Fatalf("Failed to listen on port %s: %v", cfg.GrpcPort, err)
	}
	logger.Infof("gRPC server listening on %s", cfg.GrpcPort)

	grpcServer := grpc.NewServer()

	userpb.RegisterUserServiceServer(grpcServer, userGrpcHandler)

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

	<-quit
	logger.Warn("Shutdown signal received...")

	logger.Info("Attempting graceful shutdown of gRPC server...")
	grpcServer.GracefulStop()
	logger.Info("gRPC server gracefully stopped.")
	logger.Info("User Service shut down gracefully.")
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
