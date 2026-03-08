package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/realtime-notification-system/notification-service/internal/config"
)

type Consumer interface {
	Start(ctx context.Context, handler func(ctx context.Context, message *NotificationMessage) error) error
	Close() error
}

type consumer struct {
	reader *kafka.Reader
}

func NewConsumer(cfg *config.KafkaConfig) (Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:                cfg.Brokers,
		Topic:                  cfg.TopicNotifications,
		GroupID:                cfg.ConsumerGroup,
		MinBytes:               1,
		MaxBytes:               10e6, // 10MB
		MaxWait:                1 * time.Second,
		ReadLagInterval:        -1,
		GroupBalancers:         []kafka.GroupBalancer{&kafka.RoundRobinGroupBalancer{}},
		StartOffset:            kafka.FirstOffset,
		CommitInterval:         0,
		PartitionWatchInterval: 5 * time.Second,
		MaxAttempts:            3,
	})

	return &consumer{reader: reader}, nil
}

func (c *consumer) Start(ctx context.Context, handler func(ctx context.Context, message *NotificationMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if err == context.Canceled {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		var notificationMessage NotificationMessage
		if err := json.Unmarshal(msg.Value, &notificationMessage); err != nil {
			fmt.Printf("Failed to unmarshal message: %v\n", err)
			continue
		}

		if err := handler(ctx, &notificationMessage); err != nil {
			fmt.Printf("Failed to handle message: %v\n", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			fmt.Printf("Failed to commit message: %v\n", err)
		}
	}
}

func (c *consumer) Close() error {
	return c.reader.Close()
}
