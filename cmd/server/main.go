package main

import (
	"context"
	"log"
	"net/http"
	"order-service/internal/cache"
	"order-service/internal/database"
	"order-service/internal/models"
	"order-service/internal/handler"
	"order-service/internal/kafka"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	db, err := database.New("postgres://orders_user:orders_password@localhost:5432/orders_db?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	orders, err := db.GetAllOrders()
	if err != nil {
		log.Printf("Warning: could not restore cache from database: %v", err)
		orders = make(map[string]*models.Order)
	}

	cache := cache.New()
	cache.Restore(orders)

	

	kafkaConsumer := kafka.New(
		[]string{"localhost:9092"},
		"orders",
		"order-service-group",
		db,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go kafkaConsumer.Start(ctx)

	handler := handler.New(cache, db)

	router := mux.NewRouter()
	router.HandleFunc("/order/{id}", handler.GetOrder).Methods("GET")
	router.HandleFunc("/", handler.ServeHTML).Methods("GET")
	router.HandleFunc("/debug", handler.Debug).Methods("GET")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	server := &http.Server{
		Addr:         ":8081",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Server starting on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	kafkaConsumer.Close()
	log.Println("Server stopped")
}
