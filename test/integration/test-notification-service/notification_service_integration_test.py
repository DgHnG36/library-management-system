import importlib.util
import json
import os
import sys
import types
import unittest
from types import SimpleNamespace
from unittest.mock import MagicMock, patch


SERVICE_DIR = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "..", "..", "..", "services", "notification-service")
)
MAIN_FILE = os.path.join(SERVICE_DIR, "main.py")
SRC_DIR = os.path.join(SERVICE_DIR, "src")


class FakeConfig:
    PORT = "8000"
    RABBITMQ_URL = "amqp://guest:guest@localhost:5672/"
    RABBITMQ_EXCHANGE = "order-event"
    RABBITMQ_QUEUE = "notification-queue"
    RABBITMQ_ROUTING_KEYS = ["order.created", "order.canceled", "order.status_updated"]
    USER_SVC_ADDR = "localhost:40041"
    BOOK_SVC_ADDR = "localhost:40042"


class FakeEmailService:
    def __init__(self):
        self.send_order_created = MagicMock()
        self.send_order_canceled = MagicMock()
        self.send_order_status_updated = MagicMock()


class FakeUserClient:
    def __init__(self, addr):
        self.addr = addr
        self.get_profile = MagicMock(return_value=None)
        self.close = MagicMock()


class FakeBookClient:
    def __init__(self, addr):
        self.addr = addr
        self.get_book = MagicMock(return_value=None)
        self.close = MagicMock()


class FakeURLParameters:
    def __init__(self, url):
        self.url = url
        self.heartbeat = None
        self.blocked_connection_timeout = None


class FakeBlockingConnection:
    def __init__(self, params):
        self.params = params
        self._channel = MagicMock()
        self.is_open = True

    def channel(self):
        return self._channel

    def close(self):
        self.is_open = False


class FakeAMQPConnectionError(Exception):
    pass


def load_main_module():
    src_pkg = types.ModuleType("src")
    src_pkg.__path__ = [SRC_DIR]

    src_client_pkg = types.ModuleType("src.client")
    src_client_pkg.__path__ = [os.path.join(SRC_DIR, "client")]

    src_utils_pkg = types.ModuleType("src.utils")
    src_utils_pkg.__path__ = [os.path.join(SRC_DIR, "utils")]

    email_module = types.ModuleType("src.email_service")
    email_module.EmailService = FakeEmailService

    user_client_module = types.ModuleType("src.client.user_client")
    user_client_module.UserClient = FakeUserClient

    book_client_module = types.ModuleType("src.client.book_client")
    book_client_module.BookClient = FakeBookClient

    config_module = types.ModuleType("src.utils.config")
    config_module.config = FakeConfig()

    logger_module = types.ModuleType("src.utils.logger")
    logger_module.logger = MagicMock()

    pika_module = types.ModuleType("pika")
    pika_module.URLParameters = FakeURLParameters
    pika_module.BlockingConnection = FakeBlockingConnection

    pika_exceptions_module = types.ModuleType("pika.exceptions")
    pika_exceptions_module.AMQPConnectionError = FakeAMQPConnectionError

    module_name = "notification_service_main_integration"
    if module_name in sys.modules:
        del sys.modules[module_name]

    with patch.dict(
        sys.modules,
        {
            "src": src_pkg,
            "src.client": src_client_pkg,
            "src.utils": src_utils_pkg,
            "src.email_service": email_module,
            "src.client.user_client": user_client_module,
            "src.client.book_client": book_client_module,
            "src.utils.config": config_module,
            "src.utils.logger": logger_module,
            "pika": pika_module,
            "pika.exceptions": pika_exceptions_module,
        },
    ):
        spec = importlib.util.spec_from_file_location(module_name, MAIN_FILE)
        module = importlib.util.module_from_spec(spec)
        sys.modules[module_name] = module
        spec.loader.exec_module(module)

    return module


class TestNotificationServiceIntegration(unittest.TestCase):
    def test_consumer_connect_and_bindings_success(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.connect()

        self.assertIsNotNone(consumer._connection)
        self.assertIsNotNone(consumer._channel)
        consumer._channel.exchange_declare.assert_called_once()
        consumer._channel.queue_declare.assert_called_once()
        self.assertEqual(len(FakeConfig.RABBITMQ_ROUTING_KEYS), consumer._channel.queue_bind.call_count)

    def test_consume_order_created_event_end_to_end(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@example.com", name="Alice")
        consumer.book_client.get_book.side_effect = [SimpleNamespace(title="Book A"), SimpleNamespace(title="Book B")]

        channel = MagicMock()
        method = SimpleNamespace(delivery_tag=100)
        body = json.dumps(
            {
                "event_type": "order.created",
                "payload": {
                    "user_id": "user-1",
                    "order_id": "ord-100",
                    "book_ids": ["book-1", "book-2"],
                    "due_date": "2026-04-10",
                },
            }
        ).encode()

        consumer.on_message(channel, method, None, body)

        consumer.email_service.send_order_created.assert_called_once_with(
            to_email="u@example.com",
            username="Alice",
            order_id="ord-100",
            book_titles=["Book A", "Book B"],
            due_date="2026-04-10",
        )
        channel.basic_ack.assert_called_once_with(delivery_tag=100)


if __name__ == "__main__":
    unittest.main()
