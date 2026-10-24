package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"movara/internal/config"
	"movara/internal/database"
	mysqlrepo "movara/internal/repository/mysql"
	"movara/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := database.Seed(db, os.Getenv("ADMIN_PASSWORD")); err != nil {
		log.Fatalf("seed: %v", err)
	}
	repos := mysqlrepo.New(db)

	engine, err := router.New(cfg, repos, db)
	if err != nil {
		log.Fatalf("router: %v", err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("server berjalan di http://localhost:%s (%s, data contoh)", cfg.Port, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
