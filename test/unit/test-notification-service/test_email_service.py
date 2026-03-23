import importlib
import os
import sys
import types
import unittest
from unittest.mock import MagicMock, patch


SERVICE_DIR = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "..", "..", "..", "services", "notification-service")
)
SRC_DIR = os.path.join(SERVICE_DIR, "src")
if SERVICE_DIR not in sys.path:
    sys.path.insert(0, SERVICE_DIR)


def build_src_namespace_modules():
    src_pkg = types.ModuleType("src")
    src_pkg.__path__ = [SRC_DIR]

    utils_pkg = types.ModuleType("src.utils")
    utils_pkg.__path__ = [os.path.join(SRC_DIR, "utils")]

    return src_pkg, utils_pkg


class FakeClientError(Exception):
    def __init__(self, response, operation_name):
        super().__init__(operation_name)
        self.response = response


def load_email_service_module(mock_ses_client):
    src_pkg, utils_pkg = build_src_namespace_modules()

    fake_boto3 = types.ModuleType("boto3")
    fake_boto3.client = MagicMock(return_value=mock_ses_client)

    fake_botocore = types.ModuleType("botocore")
    fake_botocore_exceptions = types.ModuleType("botocore.exceptions")
    fake_botocore_exceptions.ClientError = FakeClientError

    with patch.dict(
        sys.modules,
        {
            "src": src_pkg,
            "src.utils": utils_pkg,
            "boto3": fake_boto3,
            "botocore": fake_botocore,
            "botocore.exceptions": fake_botocore_exceptions,
        },
    ):
        module = importlib.import_module("src.email_service")
        module = importlib.reload(module)
    return module


class TestEmailService(unittest.TestCase):
    def test_send_success_returns_true(self):
        mock_ses = MagicMock()
        email_module = load_email_service_module(mock_ses)
        service = email_module.EmailService()

        ok = service.send("user@example.com", "Subject", "<p>Hello</p>")

        self.assertTrue(ok)
        mock_ses.send_email.assert_called_once()

    def test_send_client_error_returns_false(self):
        mock_ses = MagicMock()
        mock_ses.send_email.side_effect = FakeClientError({"Error": {"Message": "boom"}}, "SendEmail")
        email_module = load_email_service_module(mock_ses)
        service = email_module.EmailService()

        ok = service.send("user@example.com", "Subject", "<p>Hello</p>")

        self.assertFalse(ok)

    def test_send_order_created_delegates_send(self):
        mock_ses = MagicMock()
        email_module = load_email_service_module(mock_ses)
        service = email_module.EmailService()

        with patch.object(service, "send", return_value=True) as send_mock:
            service.send_order_created(
                to_email="u@example.com",
                username="Alice",
                order_id="ord-1",
                book_titles=["Book A", "Book B"],
                due_date="2026-03-30",
            )

        send_mock.assert_called_once()
        to_email, subject, body = send_mock.call_args.args
        self.assertEqual("u@example.com", to_email)
        self.assertEqual("Your Order Has Been Created", subject)
        self.assertIn("ord-1", body)
        self.assertIn("Book A", body)
        self.assertIn("Book B", body)

    def test_send_order_canceled_delegates_send(self):
        mock_ses = MagicMock()
        email_module = load_email_service_module(mock_ses)
        service = email_module.EmailService()

        with patch.object(service, "send", return_value=True) as send_mock:
            service.send_order_canceled(
                to_email="u@example.com",
                username="Alice",
                order_id="ord-2",
            )

        send_mock.assert_called_once()
        to_email, subject, body = send_mock.call_args.args
        self.assertEqual("u@example.com", to_email)
        self.assertEqual("Your Order Has Been Canceled", subject)
        self.assertIn("ord-2", body)

    def test_send_order_status_updated_maps_known_status(self):
        mock_ses = MagicMock()
        email_module = load_email_service_module(mock_ses)
        service = email_module.EmailService()

        with patch.object(service, "send", return_value=True) as send_mock:
            service.send_order_status_updated(
                to_email="u@example.com",
                username="Alice",
                order_id="ord-3",
                new_status="APPROVED",
            )

        send_mock.assert_called_once()
        to_email, subject, body = send_mock.call_args.args
        self.assertEqual("u@example.com", to_email)
        self.assertEqual("Your Order Status Has Been Updated", subject)
        self.assertIn("approved", body.lower())

    def test_send_order_status_updated_fallback_unknown_status(self):
        mock_ses = MagicMock()
        email_module = load_email_service_module(mock_ses)
        service = email_module.EmailService()

        with patch.object(service, "send", return_value=True) as send_mock:
            service.send_order_status_updated(
                to_email="u@example.com",
                username="Alice",
                order_id="ord-4",
                new_status="CUSTOM_STATUS",
            )

        send_mock.assert_called_once()
        _, _, body = send_mock.call_args.args
        self.assertIn("custom_status", body.lower())


if __name__ == "__main__":
    unittest.main()
