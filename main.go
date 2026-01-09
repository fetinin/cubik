package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := InitDB(ctx, cfg.ServerDBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer CloseDB(db)

	if err := RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := StartServer(ctx, db, cfg.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	})

	<-ctx.Done()
	log.Println("Shutting down...")
	wg.Wait()
}
