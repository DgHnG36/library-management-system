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

    module_name = "notification_service_main_tested"
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


class TestNotificationConsumerMain(unittest.TestCase):
    def test_connect_declares_exchange_queue_bindings_and_qos(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.connect()

        self.assertIsNotNone(consumer._connection)
        self.assertIsNotNone(consumer._channel)
        consumer._channel.exchange_declare.assert_called_once_with(
            exchange=FakeConfig.RABBITMQ_EXCHANGE,
            exchange_type="topic",
            durable=True,
        )
        consumer._channel.queue_declare.assert_called_once_with(
            queue=FakeConfig.RABBITMQ_QUEUE,
            durable=True,
        )
        self.assertEqual(
            len(FakeConfig.RABBITMQ_ROUTING_KEYS),
            consumer._channel.queue_bind.call_count,
        )
        consumer._channel.basic_qos.assert_called_once_with(prefetch_count=1)

    def test_on_message_order_created_ack(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()
        consumer._handle_order_created = MagicMock()

        channel = MagicMock()
        method = SimpleNamespace(delivery_tag=10)
        body = json.dumps({"event_type": "order.created", "payload": {"x": 1}}).encode()

        consumer.on_message(channel, method, None, body)

        consumer._handle_order_created.assert_called_once_with({"x": 1})
        channel.basic_ack.assert_called_once_with(delivery_tag=10)
        channel.basic_nack.assert_not_called()

    def test_on_message_unknown_event_ack_only(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()
        consumer._handle_order_created = MagicMock()
        consumer._handle_order_canceled = MagicMock()
        consumer._handle_order_status_updated = MagicMock()

        channel = MagicMock()
        method = SimpleNamespace(delivery_tag=11)
        body = json.dumps({"event_type": "unknown", "payload": {}}).encode()

        consumer.on_message(channel, method, None, body)

        consumer._handle_order_created.assert_not_called()
        consumer._handle_order_canceled.assert_not_called()
        consumer._handle_order_status_updated.assert_not_called()
        channel.basic_ack.assert_called_once_with(delivery_tag=11)

    def test_on_message_bad_json_nack(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        channel = MagicMock()
        method = SimpleNamespace(delivery_tag=12)

        consumer.on_message(channel, method, None, b"{bad json")

        channel.basic_nack.assert_called_once_with(delivery_tag=12, requeue=False)
        channel.basic_ack.assert_not_called()

    def test_on_message_handler_exception_nack(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()
        consumer._handle_order_created = MagicMock(side_effect=RuntimeError("boom"))

        channel = MagicMock()
        method = SimpleNamespace(delivery_tag=13)
        body = json.dumps({"event_type": "order.created", "payload": {"x": 1}}).encode()

        consumer.on_message(channel, method, None, body)

        channel.basic_nack.assert_called_once_with(delivery_tag=13, requeue=False)
        channel.basic_ack.assert_not_called()

    def test_handle_order_created_sends_email_with_books(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        user = SimpleNamespace(email="u@example.com", name="Alice")
        book1 = SimpleNamespace(title="Book A")
        book2 = SimpleNamespace(title="Book B")

        consumer.user_client.get_profile.return_value = user
        consumer.book_client.get_book.side_effect = [book1, book2]

        payload = {
            "user_id": "user-1",
            "order_id": "ord-1",
            "book_ids": ["b1", "b2"],
            "due_date": "2026-03-30",
        }

        consumer._handle_order_created(payload)

        consumer.email_service.send_order_created.assert_called_once_with(
            to_email="u@example.com",
            username="Alice",
            order_id="ord-1",
            book_titles=["Book A", "Book B"],
            due_date="2026-03-30",
        )

    def test_handle_order_created_user_missing_no_send(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = None

        consumer._handle_order_created({"user_id": "u", "order_id": "o", "book_ids": []})

        consumer.email_service.send_order_created.assert_not_called()

    def test_handle_order_canceled_sends_email(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@example.com", name="Bob")

        consumer._handle_order_canceled({"user_id": "u1", "order_id": "o1"})

        consumer.email_service.send_order_canceled.assert_called_once_with(
            to_email="u@example.com",
            username="Bob",
            order_id="o1",
        )

    def test_handle_order_canceled_user_has_no_email_no_send(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = SimpleNamespace(email="", name="Bob")

        consumer._handle_order_canceled({"user_id": "u1", "order_id": "o1"})

        consumer.email_service.send_order_canceled.assert_not_called()

    def test_handle_order_status_updated_sends_email(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@example.com", name="Bob")

        consumer._handle_order_status_updated(
            {"user_id": "u1", "order_id": "o1", "new_status": "APPROVED"}
        )

        consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="u@example.com",
            username="Bob",
            order_id="o1",
            new_status="APPROVED",
        )

    def test_handle_order_status_updated_user_missing_no_send(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = None

        consumer._handle_order_status_updated(
            {"user_id": "u1", "order_id": "o1", "new_status": "APPROVED"}
        )

        consumer.email_service.send_order_status_updated.assert_not_called()

    def test_shutdown_closes_resources(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        channel = MagicMock()
        channel.is_open = True
        connection = MagicMock()
        connection.is_open = True

        consumer._channel = channel
        consumer._connection = connection

        consumer._shutdown()

        channel.stop_consuming.assert_called_once()
        connection.close.assert_called_once()
        consumer.user_client.close.assert_called_once()
        consumer.book_client.close.assert_called_once()


if __name__ == "__main__":
    unittest.main()
