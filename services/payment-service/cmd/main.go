package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/myavarih/sre-hands-on/services/payment-service/bootstrap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := bootstrap.InitializeApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize payment-service: %v", err)
	}

	go func() {
		if err := app.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	app.Stop(shutdownCtx)
}
