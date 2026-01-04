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

	wg := sync.WaitGroup{}
	wg.Go(func() {
		if err := StartServer(ctx); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	})

	<-ctx.Done()
	log.Println("Shutting down...")
	wg.Wait()
}
