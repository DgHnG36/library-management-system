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
    SQS_QUEUE_URL = "https://sqs.ap-southeast-1.amazonaws.com/123456789/notification-queue"
    AWS_REGION = "ap-southeast-1"
    GRPC_USER_SERVICE_ADDR = "localhost:40041"
    GRPC_BOOK_SERVICE_ADDR = "localhost:40042"


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


class FakeSQSClient:
    def __init__(self):
        self.receive_message = MagicMock(return_value={"Messages": []})
        self.delete_message = MagicMock()


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

    fake_sqs_instance = FakeSQSClient()
    fake_boto3 = types.ModuleType("boto3")
    fake_boto3.client = MagicMock(return_value=fake_sqs_instance)

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
            "boto3": fake_boto3,
        },
    ):
        spec = importlib.util.spec_from_file_location(module_name, MAIN_FILE)
        module = importlib.util.module_from_spec(spec)
        sys.modules[module_name] = module
        spec.loader.exec_module(module)

    return module


def _sqs_msg(event_type: str, payload: dict, receipt_handle: str = "rh-1") -> dict:
    return {
        "Body": json.dumps({"event_type": event_type, "payload": payload}),
        "ReceiptHandle": receipt_handle,
    }


class TestNotificationConsumerMain(unittest.TestCase):
    def test_connect_creates_sqs_client(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.connect()

        self.assertIsNotNone(consumer._sqs)

    def test_process_message_order_created_returns_true(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()
        consumer._handle_order_created = MagicMock()

        result = consumer.process_message(_sqs_msg("order.created", {"x": 1}))

        consumer._handle_order_created.assert_called_once_with({"x": 1})
        self.assertTrue(result)

    def test_process_message_unknown_event_returns_true(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()
        consumer._handle_order_created = MagicMock()
        consumer._handle_order_canceled = MagicMock()
        consumer._handle_order_status_updated = MagicMock()

        result = consumer.process_message(_sqs_msg("unknown.event", {}))

        consumer._handle_order_created.assert_not_called()
        consumer._handle_order_canceled.assert_not_called()
        consumer._handle_order_status_updated.assert_not_called()
        self.assertTrue(result)

    def test_process_message_bad_json_returns_false(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        result = consumer.process_message({"Body": "{bad json", "ReceiptHandle": "rh-bad"})

        self.assertFalse(result)

    def test_process_message_handler_exception_returns_false(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()
        consumer._handle_order_created = MagicMock(side_effect=RuntimeError("boom"))

        result = consumer.process_message(_sqs_msg("order.created", {"x": 1}))

        self.assertFalse(result)

    def test_handle_order_created_sends_email_with_books(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        user = SimpleNamespace(email="u@example.com", username="Alice")
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

        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@example.com", username="Bob")

        consumer._handle_order_canceled({"user_id": "u1", "order_id": "o1"})

        consumer.email_service.send_order_canceled.assert_called_once_with(
            to_email="u@example.com",
            username="Bob",
            order_id="o1",
        )

    def test_handle_order_canceled_user_has_no_email_no_send(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = SimpleNamespace(email="", username="Bob")

        consumer._handle_order_canceled({"user_id": "u1", "order_id": "o1"})

        consumer.email_service.send_order_canceled.assert_not_called()

    def test_handle_order_status_updated_sends_email(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@example.com", username="Bob")

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

    def test_shutdown_closes_user_and_book_clients(self):
        module = load_main_module()
        consumer = module.NotificationConsumer()

        consumer._shutdown()

        consumer.user_client.close.assert_called_once()
        consumer.book_client.close.assert_called_once()


if __name__ == "__main__":
    unittest.main()

