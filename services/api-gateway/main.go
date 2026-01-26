package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/shared/env"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway at %s", httpAddr)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /trip/preview", enableCors(handleTripPreview))
	mux.HandleFunc("POST /trip/start", enableCors(handleTripStart))
	mux.HandleFunc("/ws/drivers", handleDriversWebSocket)
	mux.HandleFunc("/ws/riders", handleRidersWebSocket)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Println("API Gateway listening on ", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutDown := make(chan os.Signal, 1)

	// cmd + c = interrupt
	// sigterm = signal sent by kubernetes to terminate the app
	signal.Notify(shutDown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error starting server: %v", err)
	case sig := <-shutDown:
		log.Printf("Received signal %v, initiating shutdown", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// starts shutdown in a 10 second window
		// if it doesn't complete in time, it forces close
		if err := server.Shutdown(ctx); err != nil {
			log.Print("Graceful shutdown did not complete in 10s: ", err)
			server.Close()
		}
	}

	// 	log.Printf("Failed to start server: %v", err)
	// if err := server.ListenAndServe(); err != nil {
	// }
}
