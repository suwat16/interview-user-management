package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"example.com/cmd/usercount"
	"example.com/config"
	"example.com/internal/auth"
	"example.com/internal/users/handler"
	"example.com/internal/users/repository"
	"example.com/internal/users/service"
	"example.com/pkg/database"
	"example.com/pkg/middleware"
	"example.com/pkg/server"
	"github.com/gofiber/fiber/v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx)
	stop()
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) (runErr error) {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(ctx, cfg.Database.StartupTimeout)
	defer cancel()

	client, err := database.ConnectMongo(startupCtx, cfg.Database.URI, cfg.Database.CleanupTimeout)
	if err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Database.CleanupTimeout)
		defer cancel()
		if err := client.Disconnect(cleanupCtx); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("disconnect MongoDB: %w", err))
		}
	}()

	collection := client.Database(cfg.Database.Name).Collection(cfg.Database.Collection)
	userRepo := repository.NewUserRepository(collection)
	userService := service.NewUserService(userRepo)
	authHandler, err := auth.NewHandler(userService, cfg.Auth)
	if err != nil {
		return err
	}
	userHandler := handler.NewHandler(userService, middleware.RequireJWT([]byte(cfg.Auth.JWTSecret)))
	if err := repository.EnsureEmailIndex(startupCtx, collection); err != nil {
		return err
	}
	cancel()

	app := fiber.New()
	app.Use(middleware.LogRequest)
	middleware.RegisterHealthChecks(app, cfg.Database.HealthcheckTimeout, func(ctx context.Context) error {
		return client.Ping(ctx, nil)
	})
	authHandler.RegisterRoute(app)
	userHandler.RegisterRoute(app)

	countCtx, stopCounting := context.WithCancel(ctx)
	var countWorker sync.WaitGroup
	countWorker.Go(func() {
		usercount.LogUserCount(countCtx, cfg.UserCount, userRepo.Count)
	})
	defer func() {
		stopCounting()
		countWorker.Wait()
	}()

	return server.RunServer(ctx, app, cfg.Server)
}
