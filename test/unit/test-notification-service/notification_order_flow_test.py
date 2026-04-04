"""
notification_order_flow_test.py

Integration-style unit tests that drive the NotificationConsumer through a
complete order lifecycle, verifying that every state transition dispatches the
correct email and that failures are handled gracefully without crashing.

Order lifecycle paths exercised:
  A. Happy path  – created → approved → borrowed → returned
  B. Cancel path – created → canceled
  C. Resilience  – missing user, missing book, gRPC errors, bad payloads
"""

import importlib.util
import json
import os
import sys
import types
import unittest
from types import SimpleNamespace
from unittest.mock import MagicMock, call

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
SERVICE_DIR = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "..", "..", "..", "services", "notification-service")
)
MAIN_FILE = os.path.join(SERVICE_DIR, "main.py")
SRC_DIR = os.path.join(SERVICE_DIR, "src")


# ---------------------------------------------------------------------------
# Fake infrastructure identical to test_consumer_main.py so this file is
# completely self-contained and can be run in isolation.
# ---------------------------------------------------------------------------

class FakeConfig:
    PORT = "8000"
    RABBITMQ_URL = "amqp://guest:guest@localhost:5672/"
    RABBITMQ_EXCHANGE = "order-events"
    RABBITMQ_QUEUE = "notification-queue"
    RABBITMQ_ROUTING_KEYS = ["order.created", "order.canceled", "order.status_updated"]
    GRPC_USER_SERVICE_ADDR = "localhost:40041"
    GRPC_BOOK_SERVICE_ADDR = "localhost:40042"


class FakeEmailService:
    def __init__(self):
        self.send_order_created = MagicMock(return_value=True)
        self.send_order_canceled = MagicMock(return_value=True)
        self.send_order_status_updated = MagicMock(return_value=True)


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
        self._channel = MagicMock()
        self.is_open = True

    def channel(self):
        return self._channel

    def close(self):
        self.is_open = False


class FakeAMQPConnectionError(Exception):
    pass


def _load_consumer_module():
    """Load main.py with all external dependencies replaced by fakes."""
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

    module_name = "notification_order_flow_main"
    sys.modules.pop(module_name, None)

    with __import__("unittest.mock", fromlist=["patch"]).patch.dict(
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


def _make_consumer():
    """Return a NotificationConsumer with mocked clients / email service."""
    module = _load_consumer_module()
    return module.NotificationConsumer()


def _msg(event_type: str, payload: dict) -> bytes:
    return json.dumps({"event_type": event_type, "payload": payload}).encode()


def _channel_method(tag: int):
    channel = MagicMock()
    method = SimpleNamespace(delivery_tag=tag)
    return channel, method


# ---------------------------------------------------------------------------
# Path A: created → approved → borrowed → returned
# ---------------------------------------------------------------------------
class TestOrderHappyPath(unittest.TestCase):
    """Full borrow lifecycle for a single order."""

    def setUp(self):
        self.consumer = _make_consumer()
        self.user = SimpleNamespace(email="alice@example.com", name="Alice")
        self.book1 = SimpleNamespace(title="Clean Code")
        self.book2 = SimpleNamespace(title="The Pragmatic Programmer")

        self.consumer.user_client.get_profile.return_value = self.user
        self.consumer.book_client.get_book.side_effect = [self.book1, self.book2]

    def test_order_created_event_sends_email_with_all_books(self):
        channel, method = _channel_method(1)
        payload = {
            "user_id": "u-1",
            "order_id": "ord-1",
            "book_ids": ["b-1", "b-2"],
            "due_date": "2026-05-01",
        }

        self.consumer.on_message(channel, method, None, _msg("order.created", payload))

        self.consumer.email_service.send_order_created.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            book_titles=["Clean Code", "The Pragmatic Programmer"],
            due_date="2026-05-01",
        )
        channel.basic_ack.assert_called_once_with(delivery_tag=1)
        channel.basic_nack.assert_not_called()

    def test_order_approved_event_sends_status_update_email(self):
        channel, method = _channel_method(2)
        payload = {"user_id": "u-1", "order_id": "ord-1", "new_status": "APPROVED"}

        self.consumer.on_message(channel, method, None, _msg("order.status_updated", payload))

        self.consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            new_status="APPROVED",
        )
        channel.basic_ack.assert_called_once_with(delivery_tag=2)

    def test_order_borrowed_event_sends_status_update_email(self):
        channel, method = _channel_method(3)
        payload = {"user_id": "u-1", "order_id": "ord-1", "new_status": "BORROWED"}

        self.consumer.on_message(channel, method, None, _msg("order.status_updated", payload))

        self.consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            new_status="BORROWED",
        )
        channel.basic_ack.assert_called_once_with(delivery_tag=3)

    def test_order_returned_event_sends_status_update_email(self):
        channel, method = _channel_method(4)
        payload = {"user_id": "u-1", "order_id": "ord-1", "new_status": "RETURNED"}

        self.consumer.on_message(channel, method, None, _msg("order.status_updated", payload))

        self.consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            new_status="RETURNED",
        )
        channel.basic_ack.assert_called_once_with(delivery_tag=4)

    def test_full_lifecycle_processes_all_events_in_order(self):
        """Simulate 4 events in sequence; each must be acked, email called once per event."""
        consumer = _make_consumer()
        user = SimpleNamespace(email="alice@example.com", name="Alice")
        book = SimpleNamespace(title="Clean Code")

        consumer.user_client.get_profile.return_value = user
        consumer.book_client.get_book.return_value = book

        events = [
            ("order.created",        {"user_id": "u-1", "order_id": "o-1", "book_ids": ["b-1"], "due_date": "2026-05-01"}),
            ("order.status_updated", {"user_id": "u-1", "order_id": "o-1", "new_status": "APPROVED"}),
            ("order.status_updated", {"user_id": "u-1", "order_id": "o-1", "new_status": "BORROWED"}),
            ("order.status_updated", {"user_id": "u-1", "order_id": "o-1", "new_status": "RETURNED"}),
        ]

        for tag, (event_type, payload) in enumerate(events, start=1):
            ch, meth = _channel_method(tag)
            consumer.on_message(ch, meth, None, _msg(event_type, payload))
            ch.basic_ack.assert_called_once_with(delivery_tag=tag)
            ch.basic_nack.assert_not_called()

        self.assertEqual(1, consumer.email_service.send_order_created.call_count)
        self.assertEqual(3, consumer.email_service.send_order_status_updated.call_count)


# ---------------------------------------------------------------------------
# Path B: created → canceled
# ---------------------------------------------------------------------------
class TestOrderCancelPath(unittest.TestCase):
    """Order is canceled after creation."""

    def setUp(self):
        self.consumer = _make_consumer()
        self.user = SimpleNamespace(email="bob@example.com", name="Bob")
        self.consumer.user_client.get_profile.return_value = self.user
        self.consumer.book_client.get_book.return_value = SimpleNamespace(title="Refactoring")

    def test_order_created_then_canceled_sends_two_emails(self):
        # created
        ch1, m1 = _channel_method(10)
        self.consumer.on_message(
            ch1, m1, None,
            _msg("order.created", {"user_id": "u-2", "order_id": "o-2", "book_ids": ["b-3"], "due_date": "2026-06-01"}),
        )
        ch1.basic_ack.assert_called_once_with(delivery_tag=10)
        self.consumer.email_service.send_order_created.assert_called_once()

        # canceled
        ch2, m2 = _channel_method(11)
        self.consumer.on_message(
            ch2, m2, None,
            _msg("order.canceled", {"user_id": "u-2", "order_id": "o-2"}),
        )
        ch2.basic_ack.assert_called_once_with(delivery_tag=11)
        self.consumer.email_service.send_order_canceled.assert_called_once_with(
            to_email="bob@example.com",
            username="Bob",
            order_id="o-2",
        )

    def test_canceled_event_does_not_call_book_client(self):
        ch, m = _channel_method(12)
        self.consumer.on_message(
            ch, m, None,
            _msg("order.canceled", {"user_id": "u-2", "order_id": "o-2"}),
        )
        self.consumer.book_client.get_book.assert_not_called()


# ---------------------------------------------------------------------------
# Path C: resilience — missing user / book, gRPC failure, bad payloads
# ---------------------------------------------------------------------------
class TestOrderFlowResilience(unittest.TestCase):

    def test_created_user_not_found_no_email_and_ack(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = None

        ch, m = _channel_method(20)
        consumer.on_message(
            ch, m, None,
            _msg("order.created", {"user_id": "ghost", "order_id": "o-3", "book_ids": ["b-1"], "due_date": "N/A"}),
        )

        consumer.email_service.send_order_created.assert_not_called()
        ch.basic_ack.assert_called_once_with(delivery_tag=20)
        ch.basic_nack.assert_not_called()

    def test_created_user_email_empty_no_email_and_ack(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="", name="Ghost")

        ch, m = _channel_method(21)
        consumer.on_message(
            ch, m, None,
            _msg("order.created", {"user_id": "u-ghost", "order_id": "o-4", "book_ids": [], "due_date": "N/A"}),
        )

        consumer.email_service.send_order_created.assert_not_called()
        ch.basic_ack.assert_called_once_with(delivery_tag=21)

    def test_created_book_not_found_skips_that_title_but_still_sends_email(self):
        """If one book lookup fails, skip that title but still send email for the rest."""
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@e.com", name="User")
        # book_ids has 2 entries; second lookup returns None (not found)
        consumer.book_client.get_book.side_effect = [SimpleNamespace(title="Found Book"), None]

        ch, m = _channel_method(22)
        consumer.on_message(
            ch, m, None,
            _msg("order.created", {"user_id": "u-1", "order_id": "o-5", "book_ids": ["b-ok", "b-missing"], "due_date": "2026-07-01"}),
        )

        consumer.email_service.send_order_created.assert_called_once()
        _, kwargs = consumer.email_service.send_order_created.call_args
        self.assertEqual(["Found Book"], kwargs["book_titles"])
        ch.basic_ack.assert_called_once_with(delivery_tag=22)

    def test_status_updated_user_not_found_no_email_and_ack(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = None

        ch, m = _channel_method(23)
        consumer.on_message(
            ch, m, None,
            _msg("order.status_updated", {"user_id": "ghost", "order_id": "o-6", "new_status": "APPROVED"}),
        )

        consumer.email_service.send_order_status_updated.assert_not_called()
        ch.basic_ack.assert_called_once_with(delivery_tag=23)

    def test_canceled_user_not_found_no_email_and_ack(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = None

        ch, m = _channel_method(24)
        consumer.on_message(
            ch, m, None,
            _msg("order.canceled", {"user_id": "ghost", "order_id": "o-7"}),
        )

        consumer.email_service.send_order_canceled.assert_not_called()
        ch.basic_ack.assert_called_once_with(delivery_tag=24)

    def test_email_service_raises_does_not_propagate_and_nacks(self):
        """If email_service.send_order_created raises, consumer should nack."""
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@e.com", name="User")
        consumer.book_client.get_book.return_value = SimpleNamespace(title="A Book")
        consumer.email_service.send_order_created.side_effect = RuntimeError("SES down")

        ch, m = _channel_method(25)
        consumer.on_message(
            ch, m, None,
            _msg("order.created", {"user_id": "u-1", "order_id": "o-8", "book_ids": ["b-1"], "due_date": "N/A"}),
        )

        ch.basic_nack.assert_called_once_with(delivery_tag=25, requeue=False)
        ch.basic_ack.assert_not_called()

    def test_malformed_json_body_nacks_without_crash(self):
        consumer = _make_consumer()
        ch, m = _channel_method(26)

        consumer.on_message(ch, m, None, b"{not valid json")

        ch.basic_nack.assert_called_once_with(delivery_tag=26, requeue=False)
        ch.basic_ack.assert_not_called()

    def test_missing_event_type_field_acks_as_unknown(self):
        """A message without event_type key should be acked (treated as unknown event)."""
        consumer = _make_consumer()
        ch, m = _channel_method(27)

        consumer.on_message(ch, m, None, json.dumps({"payload": {}}).encode())

        ch.basic_ack.assert_called_once_with(delivery_tag=27)
        consumer.email_service.send_order_created.assert_not_called()

    def test_multiple_consecutive_events_each_acked_independently(self):
        """Ack/nack state of one message must not affect the next."""
        consumer = _make_consumer()
        user = SimpleNamespace(email="u@e.com", name="User")
        consumer.user_client.get_profile.return_value = user
        consumer.book_client.get_book.return_value = SimpleNamespace(title="Go Programming")

        channels = []
        for tag, (etype, payload) in enumerate(
            [
                ("order.created",        {"user_id": "u-1", "order_id": "o-seq", "book_ids": ["b-1"], "due_date": "2026-08-01"}),
                ("order.status_updated", {"user_id": "u-1", "order_id": "o-seq", "new_status": "APPROVED"}),
                ("order.canceled",       {"user_id": "u-1", "order_id": "o-seq"}),
            ],
            start=30,
        ):
            ch, m = _channel_method(tag)
            consumer.on_message(ch, m, None, _msg(etype, payload))
            channels.append((tag, ch))

        for tag, ch in channels:
            ch.basic_ack.assert_called_once_with(delivery_tag=tag)
            ch.basic_nack.assert_not_called()

    def test_overdue_status_sent_correctly(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@e.com", name="User")

        ch, m = _channel_method(40)
        consumer.on_message(
            ch, m, None,
            _msg("order.status_updated", {"user_id": "u-1", "order_id": "o-overdue", "new_status": "OVERDUE"}),
        )

        consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="u@e.com",
            username="User",
            order_id="o-overdue",
            new_status="OVERDUE",
        )
        ch.basic_ack.assert_called_once_with(delivery_tag=40)


if __name__ == "__main__":
    unittest.main()
