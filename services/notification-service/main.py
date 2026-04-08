import json
import signal
import sys
import time

import boto3

from src.client.book_client import BookClient
from src.client.user_client import UserClient
from src.email_service import EmailService
from src.utils.config import config
from src.utils.logger import logger


class NotificationConsumer:
    def __init__(self):
        self.email_service = EmailService()
        self.user_client = UserClient(config.GRPC_USER_SERVICE_ADDR)
        self.book_client = BookClient(config.GRPC_BOOK_SERVICE_ADDR)
        self._sqs = None
        self._running = False

    def connect(self):
        self._sqs = boto3.client(
            "sqs",
            region_name=config.AWS_REGION,
        )
        logger.info("Connected to SQS", extra={"queue": config.SQS_QUEUE_URL})

    def process_message(self, message: dict) -> bool:
        """Process a single SQS message. Returns True on success (caller deletes), False on failure."""
        try:
            event = json.loads(message["Body"])
            event_type = event.get("event_type")
            payload = event.get("payload", {})

            logger.info(f"Received event: {event_type}", extra={"payload": payload})

            if event_type == "order.created":
                self._handle_order_created(payload)
            elif event_type == "order.canceled":
                self._handle_order_canceled(payload)
            elif event_type == "order.status_updated":
                self._handle_order_status_updated(payload)
            else:
                logger.warning(f"Unknown event type {event_type}")

            return True

        except Exception as e:
            logger.error(f"Failed to process message: {e}", extra={"body": message.get("Body", "")})
            return False

    def _handle_order_created(self, payload: dict):
        user_id = payload.get("user_id")
        order_id = payload.get("order_id")
        book_ids = payload.get("book_ids", [])
        due_date = payload.get("due_date", "N/A")

        user = self.user_client.get_profile(user_id)
        if not user or not user.email:
            logger.warning(f"User {user_id} not found or has no email")
            return

        book_titles = []
        for book_id in book_ids:
            book = self.book_client.get_book(book_id)
            if book:
                book_titles.append(book.title)

        self.email_service.send_order_created(
            to_email=user.email,
            username=user.name,
            order_id=order_id,
            book_titles=book_titles,
            due_date=due_date
        )

    def _handle_order_canceled(self, payload: dict):
        user_id = payload.get("user_id")
        order_id = payload.get("order_id")

        user = self.user_client.get_profile(user_id)
        if not user or not user.email:
            logger.warning(f"User {user_id} not found or has no email")
            return

        self.email_service.send_order_canceled(
            to_email=user.email,
            username=user.name,
            order_id=order_id
        )

    def _handle_order_status_updated(self, payload: dict):
        user_id = payload.get("user_id")
        order_id = payload.get("order_id")
        new_status = payload.get("new_status", "")

        user = self.user_client.get_profile(user_id)
        if not user or not user.email:
            logger.warning(f"User {user_id} not found or has no email")
            return

        self.email_service.send_order_status_updated(
            to_email=user.email,
            username=user.name,
            order_id=order_id,
            new_status=new_status,
        )

    def start(self):
        self._running = True
        retry_delay = 5

        logger.info("Notification service started, waiting for events...")
        print(fr"""
             _   _  ___ _____ ___     ____  _____ ______     _____ ____ _____
            | \ | |/ _ \_   _|_ _|   / ___|| ____|  _ \ \   / /_ _/ ___| ____|
            |  \| | | | || |  | |____\___ \|  _| | |_) \ \ / / | | |   |  _|
            | |\  | |_| || |  | |_____|__) | |___|  _ < \ V /  | | |___| |___
            |_| \_|\___/ |_| |___|   |____/|_____|_| \_\ \_/  |___\____|_____|
                        Notification Service is running...
                        Port: {config.PORT}
          """)

        while self._running:
            try:
                response = self._sqs.receive_message(
                    QueueUrl=config.SQS_QUEUE_URL,
                    MaxNumberOfMessages=10,
                    WaitTimeSeconds=20,
                    VisibilityTimeout=60,
                )

                for message in response.get("Messages", []):
                    if self.process_message(message):
                        self._sqs.delete_message(
                            QueueUrl=config.SQS_QUEUE_URL,
                            ReceiptHandle=message["ReceiptHandle"],
                        )
                    else:
                        logger.warning("Message processing failed, will be requeued after visibility timeout")

            except Exception as e:
                logger.error(f"SQS polling error: {e}")
                if self._running:
                    time.sleep(retry_delay)

    def _shutdown(self):
        self._running = False
        self.user_client.close()
        self.book_client.close()
        logger.info("Notification service stopped")


consumer = NotificationConsumer()


def handle_signal(sig, frame):
    logger.info(f"Received signal {sig}, shutting down...")
    consumer._shutdown()
    sys.exit(0)


signal.signal(signal.SIGINT, handle_signal)
signal.signal(signal.SIGTERM, handle_signal)

if __name__ == "__main__":
    consumer.connect()
    consumer.start()

    def __init__(self):
        self.email_service = EmailService()
        self.user_client = UserClient(config.GRPC_USER_SERVICE_ADDR)
        self.book_client = BookClient(config.GRPC_BOOK_SERVICE_ADDR)
        self._sqs = None
        self._running = False

    def connect(self):
        self._sqs = boto3.client(
            "sqs",
            region_name=config.AWS_REGION,
        )
        logger.info("Connected to SQS", extra={"queue": config.SQS_QUEUE_URL})

    def process_message(self, message: dict) -> bool:
        """Process a single SQS message. Returns True on success (caller deletes), False on failure."""
        try:
            event = json.loads(message["Body"])
            event_type = event.get("event_type")
            payload = event.get("payload", {})

            logger.info(f"Received event: {event_type}", extra={"payload": payload})

            if event_type == "order.created":
                self._handle_order_created(payload)
            elif event_type == "order.canceled":
                self._handle_order_canceled(payload)
            elif event_type == "order.status_updated":
                self._handle_order_status_updated(payload)
            else:
                logger.warning(f"Unknown event type {event_type}")

            return True

        except Exception as e:
            logger.error(f"Failed to process message: {e}", extra={"body": message.get("Body", "")})
            return False
        
    def _handle_order_created(self, payload: dict):
        user_id = payload.get("user_id")
        order_id = payload.get("order_id")
        book_ids = payload.get("book_ids", [])
        due_date = payload.get("due_date", "N/A")
        
        user = self.user_client.get_profile(user_id)
        if not user or not user.email:
            logger.warning(f"User {user_id} not found or has no email")
            return
        
        book_titles = []
        for book_id in book_ids:
            book = self.book_client.get_book(book_id)
            if book:
                book_titles.append(book.title)
                
        self.email_service.send_order_created(
            to_email=user.email,
            username=user.name,
            order_id=order_id,
            book_titles=book_titles,
            due_date=due_date
        )
        
    def _handle_order_canceled(self, payload: dict):
        user_id = payload.get("user_id")
        order_id = payload.get("order_id")
        
        user = self.user_client.get_profile(user_id)
        if not user or not user.email:
            logger.warning(f"User {user_id} not found or has no email")
            return
        
        self.email_service.send_order_canceled(
            to_email=user.email,
            username=user.name,
            order_id=order_id
        )
        
    def _handle_order_status_updated(self, payload: dict):
        user_id = payload.get("user_id")
        order_id = payload.get("order_id")
        new_status = payload.get("new_status", "")

        user = self.user_client.get_profile(user_id)
        if not user or not user.email:
            logger.warning(f"User {user_id} not found or has no email")
            return

        self.email_service.send_order_status_updated(
            to_email=user.email,
            username=user.name,
            order_id=order_id,
            new_status=new_status,
        )
        
    def start(self):
        self._running = True
        retry_delay = 5

        logger.info("Notification service started, waiting for events...")
        print(fr"""
             _   _  ___ _____ ___     ____  _____ ______     _____ ____ _____
            | \ | |/ _ \_   _|_ _|   / ___|| ____|  _ \ \   / /_ _/ ___| ____|
            |  \| | | | || |  | |____\___ \|  _| | |_) \ \ / / | | |   |  _|
            | |\  | |_| || |  | |_____|__) | |___|  _ < \ V /  | | |___| |___
            |_| \_|\___/ |_| |___|   |____/|_____|_| \_\ \_/  |___\____|_____|
                        Notification Service is running...
                        Port: {config.PORT}
          """)

        while self._running:
            try:
                response = self._sqs.receive_message(
                    QueueUrl=config.SQS_QUEUE_URL,
                    MaxNumberOfMessages=10,
                    WaitTimeSeconds=20,
                    VisibilityTimeout=60,
                )

                for message in response.get("Messages", []):
                    if self.process_message(message):
                        self._sqs.delete_message(
                            QueueUrl=config.SQS_QUEUE_URL,
                            ReceiptHandle=message["ReceiptHandle"],
                        )
                    else:
                        logger.warning("Message processing failed, will be requeued after visibility timeout")

            except Exception as e:
                logger.error(f"SQS polling error: {e}")
                if self._running:
                    time.sleep(retry_delay)

    def _shutdown(self):
        self._running = False
        self.user_client.close()
        self.book_client.close()
        logger.info("Notification service stopped")


consumer = NotificationConsumer()


def handle_signal(sig, frame):
    logger.info(f"Received signal {sig}, shutting down...")
    consumer._shutdown()
    sys.exit(0)


signal.signal(signal.SIGINT, handle_signal)
signal.signal(signal.SIGTERM, handle_signal)

if __name__ == "__main__":
    consumer.connect()
    consumer.start()