package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"taskmanager/internal/config"
	"taskmanager/internal/db"
	"taskmanager/internal/handlers"
	"taskmanager/internal/migrations"
	"taskmanager/internal/repository"
	"taskmanager/internal/router"
	"taskmanager/internal/sse"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	activityRepo := repository.NewActivityRepository(pool)
	attachmentRepo := repository.NewAttachmentRepository(pool)

	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret)
	hub := sse.NewHub()
	taskHandler := handlers.NewTaskHandler(taskRepo, activityRepo, hub)
	attachmentHandler := handlers.NewAttachmentHandler(taskRepo, attachmentRepo, cfg.StorageDir)

	r := router.New(router.Deps{
		JWTSecret:         cfg.JWTSecret,
		AllowedOrigin:     cfg.AllowedOrigin,
		AuthHandler:       authHandler,
		TaskHandler:       taskHandler,
		AttachmentHandler: attachmentHandler,
	})

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
