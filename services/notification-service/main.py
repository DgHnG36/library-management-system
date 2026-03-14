import json
import signal
import sys
import time

import pika
import pika.exceptions

from src.client.book_client import BookClient
from src.client.user_client import UserClient
from src.email_service import EmailService
from src.utils.config import config
from src.utils.logger import logger

class NotificationConsumer:
    def __init__(self):
        self.email_service = EmailService()
        self.user_client = UserClient(config.USER_SVC_ADDR)
        self.book_client = BookClient(config.BOOK_SVC_ADDR)
        self._connection = None
        self._channel = None
    
    def connect(self):
        params = pika.URLParameters(config.RABBITMQ_URL)
        params.heartbeat = 60
        params.blocked_connection_timeout = 300
        
        self._connection = pika.BlockingConnection(params)
        self._channel = self._connection.channel()
        
        self._channel.exchange_declare(
            exchange=config.NOTIFICATION_EXCHANGE,
            exchange_type="topic",
            durable=True
        )
        
        self._channel.queue_declare(
            queue=config.RABBITMQ_QUEUE,
            durable=True
        )
        
        for routing_key in config.RABBITMQ_ROUTING_KEYS:
            self._channel.queue_bind(
                exchange=config.NOTIFICATION_EXCHANGE,
                queue=config.RABBITMQ_QUEUE,
                routing_key=routing_key
            )
            
        self._channel.basic_qos(prefetch_count=1)
        logger.info("Connect to RabbitMQ successfully", extra={
            "exchange": config.RABBITMQ_EXCHANGE,
            "queue": config.RABBITMQ_QUEUE,
        })
        
    def on_message(self, channel, method, properties, body):
        try:
            event = json.loads(body)
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
                
            channel.basic_ack(delivery_tag=method.delivery_tag)
            
        except Exception as e:
            logger.error(f"Failed to process message: {e}", extra={"body": body.decode()})
            channel.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
        
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
            user_name=user.name,
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
            user_name=user.name,
            order_id=order_id
        )
        
    def _handle_order_status_updated(self, payload: dict):
        user_id = payload.get("user_id")
        order_id = payload.get("order_id")
        new_status = payload.get("new_status", "")

        user = self._user_client.get_profile(user_id)
        if not user or not user.email:
            logger.warning(f"User {user_id} not found or has no email")
            return

        self._email_service.send_order_status_updated(
            to_email=user.email,
            username=user.username,
            order_id=order_id,
            new_status=new_status,
        )
        
    def start(self):
        retry_delay = 5
        
        while True:
            try:    
                self.connect()
                self._channel.basic_consume(
                    queue=config.RABBITMQ_QUEUE,
                    on_message_callback=self.on_message
                )
                logger.info("Notification service started, waiting for events...")
                print(f"""
                     _   _  ___ _____ ___     ____  _____ ______     _____ ____ _____ 
                    | \ | |/ _ \_   _|_ _|   / ___|| ____|  _ \ \   / /_ _/ ___| ____|
                    |  \| | | | || |  | |____\___ \|  _| | |_) \ \ / / | | |   |  _|  
                    | |\  | |_| || |  | |_____|__) | |___|  _ < \ V /  | | |___| |___ 
                    |_| \_|\___/ |_| |___|   |____/|_____|_| \_\ \_/  |___\____|_____|
                                Notification Service is running...
                                Port: {config.PORT}
                      """)
                self._channel.start_consuming()
            except pika.exceptions.AMQPConnectionError as e:
                logger.error(f"RabbitMQ connection lost, retrying in {retry_delay}s: {e}")
                time.sleep(retry_delay)
            except KeyboardInterrupt:
                logger.info("Shutting down notification service...")
                self._shutdown()
                break
            
    def _shutdown(self):
        if self._channel and self._channel.is_open:
            self._channel.stop_consuming()
        if self._connection and self._connection.is_open:
            self._connection.close()
        self._user_client.close()
        self._book_client.close()
        logger.info("Notification service stopped")

consumer = NotificationConsumer()
def handle_signal(sig, frame):
    logger.info(f"Received signal {sig}, shutting down...")
    consumer._shutdown()
    sys.exit(0)
    
signal.signal(signal.SIGINT, handle_signal)
signal.signal(signal.SIGTERM, handle_signal)

if __name__ == "__main__":
    consumer.start()