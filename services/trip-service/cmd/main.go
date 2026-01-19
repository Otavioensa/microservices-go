package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	h "ride-sharing/services/trip-service/internal/infrastructure/http"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/env"
	"syscall"
	"time"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8083")
)

func main() {
	log.Println("Starting Trip service at %v", httpAddr)
	inmemRepo := repository.NewInMemRepository()
	svc := service.NewService(inmemRepo)

	mux := http.NewServeMux()

	handler := &h.HttpHandler{Service: svc}

	mux.HandleFunc("POST /preview", handler.HandleTripPreview)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Println("Trip service listening on ", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	// cmd + c = interrupt
	// sigterm = signal sent by kubernetes to terminate the app
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Failed to start server: %v", err)
	case sign := <-shutdown:
		log.Printf("Received signal %v, initiating shutdown", sign)
		// timeout of 10 seconds
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Print("Graceful shutdown did not complete in 10s: ", err)
			server.Close()
		}
	}

	// if err := server.ListenAndServe(); err != nil {
	// 	log.Printf("Failed to start server: %v", err)
	// }
}
