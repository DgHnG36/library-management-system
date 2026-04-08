import os


class Config:
    PORT: str = os.getenv("PORT", "8000")
    SQS_QUEUE_URL: str = os.getenv("SQS_QUEUE_URL", "")

    GRPC_USER_SERVICE_ADDR: str = os.getenv(
        "GRPC_USER_SERVICE_ADDR", os.getenv("USER_SVC_ADDR", "localhost:40041")
    )
    GRPC_BOOK_SERVICE_ADDR: str = os.getenv(
        "GRPC_BOOK_SERVICE_ADDR", os.getenv("BOOK_SVC_ADDR", "localhost:40042")
    )

    AWS_REGION: str = os.getenv("AWS_REGION", "ap-southeast-1")
    AWS_ACCESS_KEY_ID: str = os.getenv("AWS_ACCESS_KEY_ID", "")
    AWS_SECRET_ACCESS_KEY: str = os.getenv("AWS_SECRET_ACCESS_KEY", "")
    SES_SENDER_EMAIL: str = os.getenv("SES_SENDER_EMAIL", "noreply@yourdomain.com")

    RABBITMQ_URL: str = os.getenv("RABBITMQ_URL", "")
    RABBITMQ_EXCHANGE: str = os.getenv("RABBITMQ_EXCHANGE", "order-events")
    RABBITMQ_QUEUE: str = os.getenv("RABBITMQ_QUEUE", "notification-queue")

    ENV: str = os.getenv("ENV", "development")


config = Config()
