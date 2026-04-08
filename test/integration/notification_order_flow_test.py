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
from unittest.mock import MagicMock

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
SERVICE_DIR = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "..", "..", "services", "notification-service")
)
MAIN_FILE = os.path.join(SERVICE_DIR, "main.py")
SRC_DIR = os.path.join(SERVICE_DIR, "src")


# ---------------------------------------------------------------------------
# Fake infrastructure — completely self-contained, no real AWS/SQS calls.
# ---------------------------------------------------------------------------

class FakeConfig:
    PORT = "8000"
    SQS_QUEUE_URL = "https://sqs.ap-southeast-1.amazonaws.com/123456789/notification-queue"
    AWS_REGION = "ap-southeast-1"
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


class FakeSQSClient:
    def __init__(self):
        self.receive_message = MagicMock(return_value={"Messages": []})
        self.delete_message = MagicMock()


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

    fake_sqs_instance = FakeSQSClient()
    fake_boto3 = types.ModuleType("boto3")
    fake_boto3.client = MagicMock(return_value=fake_sqs_instance)

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
            "boto3": fake_boto3,
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


def _msg(event_type: str, payload: dict, receipt_handle: str = "fake-receipt") -> dict:
    """Build a fake SQS message dict."""
    return {
        "Body": json.dumps({"event_type": event_type, "payload": payload}),
        "ReceiptHandle": receipt_handle,
    }


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
        payload = {
            "user_id": "u-1",
            "order_id": "ord-1",
            "book_ids": ["b-1", "b-2"],
            "due_date": "2026-05-01",
        }

        result = self.consumer.process_message(_msg("order.created", payload))

        self.assertTrue(result)
        self.consumer.email_service.send_order_created.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            book_titles=["Clean Code", "The Pragmatic Programmer"],
            due_date="2026-05-01",
        )

    def test_order_approved_event_sends_status_update_email(self):
        payload = {"user_id": "u-1", "order_id": "ord-1", "new_status": "APPROVED"}

        result = self.consumer.process_message(_msg("order.status_updated", payload))

        self.assertTrue(result)
        self.consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            new_status="APPROVED",
        )

    def test_order_borrowed_event_sends_status_update_email(self):
        payload = {"user_id": "u-1", "order_id": "ord-1", "new_status": "BORROWED"}

        result = self.consumer.process_message(_msg("order.status_updated", payload))

        self.assertTrue(result)
        self.consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            new_status="BORROWED",
        )

    def test_order_returned_event_sends_status_update_email(self):
        payload = {"user_id": "u-1", "order_id": "ord-1", "new_status": "RETURNED"}

        result = self.consumer.process_message(_msg("order.status_updated", payload))

        self.assertTrue(result)
        self.consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="alice@example.com",
            username="Alice",
            order_id="ord-1",
            new_status="RETURNED",
        )

    def test_full_lifecycle_processes_all_events_in_order(self):
        """Simulate 4 events in sequence; each must succeed, email called once per event."""
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

        for event_type, payload in events:
            result = consumer.process_message(_msg(event_type, payload))
            self.assertTrue(result)

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
        result1 = self.consumer.process_message(
            _msg("order.created", {"user_id": "u-2", "order_id": "o-2", "book_ids": ["b-3"], "due_date": "2026-06-01"})
        )
        self.assertTrue(result1)
        self.consumer.email_service.send_order_created.assert_called_once()

        result2 = self.consumer.process_message(
            _msg("order.canceled", {"user_id": "u-2", "order_id": "o-2"})
        )
        self.assertTrue(result2)
        self.consumer.email_service.send_order_canceled.assert_called_once_with(
            to_email="bob@example.com",
            username="Bob",
            order_id="o-2",
        )

    def test_canceled_event_does_not_call_book_client(self):
        self.consumer.process_message(
            _msg("order.canceled", {"user_id": "u-2", "order_id": "o-2"})
        )
        self.consumer.book_client.get_book.assert_not_called()


# ---------------------------------------------------------------------------
# Path C: resilience — missing user / book, gRPC failure, bad payloads
# ---------------------------------------------------------------------------
class TestOrderFlowResilience(unittest.TestCase):

    def test_created_user_not_found_no_email_and_success(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = None

        result = consumer.process_message(
            _msg("order.created", {"user_id": "ghost", "order_id": "o-3", "book_ids": ["b-1"], "due_date": "N/A"})
        )

        consumer.email_service.send_order_created.assert_not_called()
        self.assertTrue(result)

    def test_created_user_email_empty_no_email_and_success(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="", name="Ghost")

        result = consumer.process_message(
            _msg("order.created", {"user_id": "u-ghost", "order_id": "o-4", "book_ids": [], "due_date": "N/A"})
        )

        consumer.email_service.send_order_created.assert_not_called()
        self.assertTrue(result)

    def test_created_book_not_found_skips_that_title_but_still_sends_email(self):
        """If one book lookup fails, skip that title but still send email for the rest."""
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@e.com", name="User")
        consumer.book_client.get_book.side_effect = [SimpleNamespace(title="Found Book"), None]

        result = consumer.process_message(
            _msg("order.created", {"user_id": "u-1", "order_id": "o-5", "book_ids": ["b-ok", "b-missing"], "due_date": "2026-07-01"})
        )

        self.assertTrue(result)
        consumer.email_service.send_order_created.assert_called_once()
        _, kwargs = consumer.email_service.send_order_created.call_args
        self.assertEqual(["Found Book"], kwargs["book_titles"])

    def test_status_updated_user_not_found_no_email_and_success(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = None

        result = consumer.process_message(
            _msg("order.status_updated", {"user_id": "ghost", "order_id": "o-6", "new_status": "APPROVED"})
        )

        consumer.email_service.send_order_status_updated.assert_not_called()
        self.assertTrue(result)

    def test_canceled_user_not_found_no_email_and_success(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = None

        result = consumer.process_message(
            _msg("order.canceled", {"user_id": "ghost", "order_id": "o-7"})
        )

        consumer.email_service.send_order_canceled.assert_not_called()
        self.assertTrue(result)

    def test_email_service_raises_returns_false(self):
        """If email_service raises, process_message should return False (SQS will redeliver)."""
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@e.com", name="User")
        consumer.book_client.get_book.return_value = SimpleNamespace(title="A Book")
        consumer.email_service.send_order_created.side_effect = RuntimeError("SES down")

        result = consumer.process_message(
            _msg("order.created", {"user_id": "u-1", "order_id": "o-8", "book_ids": ["b-1"], "due_date": "N/A"})
        )

        self.assertFalse(result)

    def test_malformed_json_body_returns_false(self):
        consumer = _make_consumer()

        result = consumer.process_message({"Body": "{not valid json", "ReceiptHandle": "rh-bad"})

        self.assertFalse(result)

    def test_missing_event_type_field_returns_true_as_unknown(self):
        """A message without event_type should be processed as unknown — returns True."""
        consumer = _make_consumer()

        result = consumer.process_message(
            {"Body": json.dumps({"payload": {}}), "ReceiptHandle": "rh-1"}
        )

        self.assertTrue(result)
        consumer.email_service.send_order_created.assert_not_called()

    def test_multiple_consecutive_events_each_succeed_independently(self):
        """Each message is independent; one failure must not affect the next."""
        consumer = _make_consumer()
        user = SimpleNamespace(email="u@e.com", name="User")
        consumer.user_client.get_profile.return_value = user
        consumer.book_client.get_book.return_value = SimpleNamespace(title="Go Programming")

        events = [
            ("order.created",        {"user_id": "u-1", "order_id": "o-seq", "book_ids": ["b-1"], "due_date": "2026-08-01"}),
            ("order.status_updated", {"user_id": "u-1", "order_id": "o-seq", "new_status": "APPROVED"}),
            ("order.canceled",       {"user_id": "u-1", "order_id": "o-seq"}),
        ]

        for event_type, payload in events:
            result = consumer.process_message(_msg(event_type, payload))
            self.assertTrue(result)

    def test_overdue_status_sent_correctly(self):
        consumer = _make_consumer()
        consumer.user_client.get_profile.return_value = SimpleNamespace(email="u@e.com", name="User")

        result = consumer.process_message(
            _msg("order.status_updated", {"user_id": "u-1", "order_id": "o-overdue", "new_status": "OVERDUE"})
        )

        self.assertTrue(result)
        consumer.email_service.send_order_status_updated.assert_called_once_with(
            to_email="u@e.com",
            username="User",
            order_id="o-overdue",
            new_status="OVERDUE",
        )


if __name__ == "__main__":
    unittest.main()



