import os

class Config:
    PORT: str = os.getenv("PORT", "8000")
    RABBITMQ_URL: str = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
    RABBITMQ_EXCHANGE: str = os.getenv("RABBITMQ_EXCHANGE", "order-event")
    RABBITMQ_QUEUE: str = os.getenv("RABBITMQ_QUEUE", "notification-queue")
    RABBITMQ_ROUTING_KEYS: list[str] = ["order.created", "order.canceled", "order.status_updated"]

    GRPC_USER_SERVICE_ADDR: str = os.getenv("USER_SVC_ADDR", "localhost:40041")
    GRPC_BOOK_SERVICE_ADDR: str = os.getenv("BOOK_SVC_ADDR", "localhost:40042")
    
    AWS_REGION: str = os.getenv("AWS_REGION", "ap-southeast-1")
    AWS_ACCESS_KEY_ID: str = os.getenv("AWS_ACCESS_KEY_ID", "")
    AWS_SECRET_ACCESS_KEY: str = os.getenv("AWS_SECRET_ACCESS_KEY", "")
    SES_SENDER_EMAIL: str = os.getenv("SES_SENDER_EMAIL", "noreply@yourdomain.com")
    
    ENV: str = os.getenv("ENV", "development")
    
config = Config()