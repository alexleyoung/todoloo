package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexyoung/todoloo/internal/api"
	"github.com/alexyoung/todoloo/internal/config"
	"github.com/alexyoung/todoloo/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		if err := database.Ping(ctx); err != nil {
			log.Printf("database ping failed: %v", err)
		}
	}()

	router := api.Router(database)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	go func() {
		log.Printf("server starting on %s", addr)
		if err := api.Serve(addr, router); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown complete")
}
