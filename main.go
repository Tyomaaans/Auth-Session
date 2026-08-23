package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auth-session/internal/config"
	"auth-session/internal/infrastructure/postgres"
	"auth-session/internal/infrastructure/redis"
	"auth-session/internal/middleware"
	"auth-session/internal/router"
	"auth-session/internal/sessions"
	"auth-session/internal/tokens"
	"auth-session/internal/users"
	"auth-session/pkg"
)

func main() {
	cfg := config.NewConfig()
	validate := pkg.New()

	redisClient := redis.NewRedisClient(cfg.REDISaddr, cfg.REDISpassword)

	db, err := postgres.NewPostgresDB(cfg.DSN)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}

	sessionSvc := sessions.NewSessionService(redisClient)
	tokenSvc := tokens.NewJWTService(cfg.JWTSecretKey, cfg.JWTExpiry, cfg.DefaultRefreshExpiry, cfg.ShortRefreshExpiry, redisClient, sessionSvc)

	userRepo := users.NewUserRepository(db)
	userService := users.NewUserService(userRepo, tokenSvc, sessionSvc, validate)
	userHandler := users.NewUserHandler(userService, cfg.DefaultRefreshExpiry, cfg.ShortRefreshExpiry)

	authMiddleware := middleware.NewAuthMiddleware(tokenSvc, cfg.AdminSecretKey)

	r := routes.NewUserRouter(userHandler, authMiddleware)

	srv := &http.Server{
		Addr:         ":" + cfg.APPport,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("Server running on :%s", cfg.APPport)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := redisClient.Close(); err != nil {
		log.Printf("Redis close error: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("DB close error: %v", err)
		}
	}

	log.Println("Server exited")
}