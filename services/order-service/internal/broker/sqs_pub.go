package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSPublisher struct {
	client   *sqs.Client
	queueURL string
	logger   *logger.Logger
}

func NewSQSPublisher(region, queueURL, accessKeyID, secretAccessKey string, log *logger.Logger) (*SQSPublisher, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &SQSPublisher{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
		logger:   log,
	}, nil
}

func (p *SQSPublisher) Publish(routingKey string, payload map[string]interface{}) error {
	msg := EventMessage{
		EventType:  routingKey,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Payload:    payload,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = p.client.SendMessage(context.Background(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		p.logger.Error("failed to publish SQS message", err, logger.Fields{
			"routing_key": routingKey,
			"queue_url":   p.queueURL,
		})
		return fmt.Errorf("failed to send SQS message: %w", err)
	}

	p.logger.Info("Event published to SQS", logger.Fields{
		"routing_key": routingKey,
		"queue_url":   p.queueURL,
	})

	return nil
}

func (p *SQSPublisher) Close() {
	// SQS client is stateless, nothing to close
}
