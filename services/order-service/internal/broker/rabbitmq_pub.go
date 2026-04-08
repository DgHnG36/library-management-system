package broker

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	Publish(routingKey string, payload map[string]interface{}) error
	Close()
}

type EventMessage struct {
	EventType  string                 `json:"event_type"`
	OccurredAt string                 `json:"occurred_at"`
	Payload    map[string]interface{} `json:"payload"`
}

type RabbitMQPublisher struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	amqpURL      string
	logger       *logger.Logger
	mu           sync.Mutex
}

func NewRabbitMQPublisher(amqpURL, exchangeName string, logger *logger.Logger) (*RabbitMQPublisher, error) {
	p := &RabbitMQPublisher{
		exchangeName: exchangeName,
		amqpURL:      amqpURL,
		logger:       logger,
	}

	if err := p.connect(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *RabbitMQPublisher) connect() error {
	conn, err := amqp.Dial(p.amqpURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		p.exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	p.channel = ch
	p.conn = conn
	return nil
}

func (p *RabbitMQPublisher) Publish(routingKey string, payload map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	msg := EventMessage{
		EventType:  routingKey,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Payload:    payload,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = p.channel.Publish(
		p.exchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)

	if err != nil {
		p.logger.Error("failed to publish message", err, logger.Fields{
			"routing_key": routingKey,
			"payload":     payload,
		})

		if reconnErr := p.connect(); reconnErr != nil {
			return fmt.Errorf("reconnect failed: %w", reconnErr)
		}

		err = p.channel.Publish(
			p.exchangeName,
			routingKey,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Timestamp:    time.Now().UTC(),
				Body:         body,
			},
		)
		if err != nil {
			return fmt.Errorf("publish retry failed: %w", err)
		}
	}

	p.logger.Info("Event published", logger.Fields{
		"routing_key": routingKey,
		"exchange":    p.exchangeName,
	})

	return nil
}

func (p *RabbitMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}
