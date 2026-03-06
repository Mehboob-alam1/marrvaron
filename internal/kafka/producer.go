package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"marvaron/internal/config"

	"github.com/segmentio/kafka-go"
)

var writers map[string]*kafka.Writer

func Init() {
	writers = make(map[string]*kafka.Writer)

	// Crea writer per ogni topic
	topics := []string{
		config.AppConfig.Kafka.TopicQRScans,
		config.AppConfig.Kafka.TopicOrders,
		config.AppConfig.Kafka.TopicInventory,
	}

	for _, topic := range topics {
		writers[topic] = &kafka.Writer{
			Addr:     kafka.TCP(config.AppConfig.Kafka.Brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
	}

	log.Println("Kafka producers initialized")
}

func Close() {
	for _, writer := range writers {
		if err := writer.Close(); err != nil {
			log.Printf("Error closing Kafka writer: %v", err)
		}
	}
}

// PublishQRScan pubblica un evento di scansione QR
func PublishQRScan(event interface{}) error {
	return publish(config.AppConfig.Kafka.TopicQRScans, event)
}

// PublishOrder pubblica un evento di ordine
func PublishOrder(event interface{}) error {
	return publish(config.AppConfig.Kafka.TopicOrders, event)
}

// PublishInventory pubblica un evento di inventario
func PublishInventory(event interface{}) error {
	return publish(config.AppConfig.Kafka.TopicInventory, event)
}

func publish(topic string, event interface{}) error {
	writer, exists := writers[topic]
	if !exists {
		return fmt.Errorf("writer for topic %s not found", topic)
	}

	// Serializza evento
	message, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Pubblica messaggio
	err = writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: message,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}
