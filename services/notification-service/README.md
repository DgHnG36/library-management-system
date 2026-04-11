# Notification Service

The **Notification Service** is an event-driven Python service that consumes order events from RabbitMQ and dispatches email notifications to users via AWS SES (or a compatible SMTP provider).

**Language:** Python | **Trigger:** RabbitMQ consumer

---

## Responsibilities

- Consume messages from the `order-events` RabbitMQ exchange
- Send transactional emails (order confirmed, order cancelled, etc.)
- Email delivery via AWS SES (boto3)

---

## Project Structure

```
notification-service/
├── src/
│   ├── client/             # RabbitMQ consumer client
│   ├── email_service.py    # Email dispatch logic (AWS SES)
│   └── utils/              # Shared utilities
├── main.py                 # Service entrypoint
├── requirements.txt        # Python dependencies
├── .env.example            # Environment variable template
└── Dockerfile
```

---

## Key Dependencies

| Package                   | Purpose                      |
| ------------------------- | ---------------------------- |
| `pika`                    | RabbitMQ AMQP client         |
| `boto3`                   | AWS SDK — SES email delivery |
| `grpcio` / `grpcio-tools` | gRPC stubs for shared types  |
| `protobuf`                | Protobuf runtime             |

---

## Configuration

Copy `.env.example` to `.env` and fill in the values:

| Variable                | Description                         |
| ----------------------- | ----------------------------------- |
| `RABBITMQ_URL`          | RabbitMQ AMQP connection string     |
| `RABBITMQ_EXCHANGE`     | Exchange to consume from            |
| `AWS_ACCESS_KEY_ID`     | AWS credentials for SES             |
| `AWS_SECRET_ACCESS_KEY` | AWS credentials for SES             |
| `AWS_REGION`            | AWS region (e.g., `ap-southeast-1`) |
| `SES_SENDER_EMAIL`      | Verified sender email address       |

---

## Running Locally

```bash
docker-compose up -d rabbitmq

cd services/notification-service
pip install -r requirements.txt
python main.py
```

---

## See Also

- [services/](../) — All services overview
- [services/order-service/](../order-service/) — Publishes events consumed here
- [docs/aws-managed-services.md](../../docs/aws-managed-services.md) — AWS SES setup guide
