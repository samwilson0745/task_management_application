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

	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret)
	taskHandler := handlers.NewTaskHandler(taskRepo)

	r := router.New(router.Deps{
		JWTSecret:     cfg.JWTSecret,
		AllowedOrigin: cfg.AllowedOrigin,
		AuthHandler:   authHandler,
		TaskHandler:   taskHandler,
	})

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
