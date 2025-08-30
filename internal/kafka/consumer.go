package kafka

import (
	"context"
	"encoding/json"
	"log"
	"order-service/internal/database"
	"order-service/internal/models"
	"sync"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	db     *database.Postgres
	cache  map[string]*models.Order
	mu     sync.Mutex
}

func New(brokers []string, topic string, groupID string, db *database.Postgres) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader: reader,
		db:     db,
		cache:  make(map[string]*models.Order),
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("Starting Kafka consumer...")

	for {
		select {
		case <-ctx.Done():
			c.reader.Close()
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			log.Printf("Received message: %s", string(msg.Value))

			var order models.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			if err := c.db.SaveOrder(&order); err != nil {
				log.Printf("Error saving order to database: %v", err)
				continue
			}
			
			c.mu.Lock()
			c.cache[order.OrderUID] = &order
			c.mu.Unlock()

			log.Printf("Order %s processed successfully", order.OrderUID)
		}
	}
}

func (c *Consumer) GetFromCache(orderUID string) (*models.Order, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	order, exists := c.cache[orderUID]
	return order, exists
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}