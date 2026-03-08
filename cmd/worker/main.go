package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/realtime-notification-system/notification-service/internal/config"
	"github.com/realtime-notification-system/notification-service/internal/database"
	"github.com/realtime-notification-system/notification-service/internal/kafka"
	"github.com/realtime-notification-system/notification-service/internal/repository"
	"github.com/realtime-notification-system/notification-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	producer, err := kafka.NewProducer(&cfg.Kafka)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	userRepo := repository.NewUserRepository(db.DB)
	notificationRepo := repository.NewNotificationRepository(db.DB)
	deliveryLogRepo := repository.NewDeliveryLogRepository(db.DB)

	deliveryService := service.NewDeliveryService(notificationRepo, deliveryLogRepo, userRepo, producer, cfg.Worker.RetryAttempts)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < cfg.Worker.Consumers; i++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()
			runConsumer(ctx, cfg, deliveryService, consumerID)
		}(i)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down workers...")
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for workers to stop")
	}

	log.Println("Worker service exited")
}

func runConsumer(ctx context.Context, cfg *config.Config, deliveryService service.DeliveryService, consumerID int) {
	consumer, err := kafka.NewConsumer(&cfg.Kafka)
	if err != nil {
		log.Printf("Failed to create consumer %d: %v", consumerID, err)
		return
	}
	defer consumer.Close()

	log.Printf("Consumer %d started", consumerID)

	handler := func(ctx context.Context, message *kafka.NotificationMessage) error {
		return deliveryService.ProcessNotification(ctx, message)
	}

	if err := consumer.Start(ctx, handler); err != nil {
		if err == context.Canceled {
			log.Printf("Consumer %d stopped", consumerID)
		} else {
			log.Printf("Consumer %d error: %v", consumerID, err)
		}
	}
}
