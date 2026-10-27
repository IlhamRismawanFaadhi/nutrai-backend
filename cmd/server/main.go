package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // embed tz database agar LoadLocation selalu bekerja

	"github.com/joho/godotenv"

	"nutrai/internal/config"
	"nutrai/internal/database"
	"nutrai/internal/router"
)

func main() {
	_ = godotenv.Load() // abaikan jika .env tidak ada (Railway pakai env vars)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	db, err := database.Connect(cfg.DSN)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if cfg.AutoMigrate {
		if err := database.Migrate(db); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	if err := database.SeedFoods(db); err != nil {
		log.Printf("seed foods failed (pastikan tabel foods ada): %v", err)
	}

	// WriteTimeout harus lebih besar dari worst-case panggilan n8n (timeout x (retry+1)).
	writeTimeout := cfg.N8NTimeout*time.Duration(cfg.N8NMaxRetries+1) + 15*time.Second

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(cfg, db),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("NutrAI API listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
