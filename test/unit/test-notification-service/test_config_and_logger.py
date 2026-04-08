import importlib
import os
import sys
import types
import unittest


SERVICE_DIR = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "..", "..", "..", "services", "notification-service")
)
SRC_DIR = os.path.join(SERVICE_DIR, "src")
if SERVICE_DIR not in sys.path:
    sys.path.insert(0, SERVICE_DIR)


def prepare_src_namespace():
    src_pkg = types.ModuleType("src")
    src_pkg.__path__ = [SRC_DIR]

    utils_pkg = types.ModuleType("src.utils")
    utils_pkg.__path__ = [os.path.join(SRC_DIR, "utils")]

    sys.modules["src"] = src_pkg
    sys.modules["src.utils"] = utils_pkg


prepare_src_namespace()


class TestConfigAndLogger(unittest.TestCase):
    def test_config_defaults(self):
        module = importlib.import_module("src.utils.config")
        cfg = module.config

        self.assertEqual("8000", cfg.PORT)
        self.assertEqual("", cfg.SQS_QUEUE_URL)
        self.assertEqual("ap-southeast-1", cfg.AWS_REGION)

    def test_logger_returns_same_named_logger(self):
        logger_module = importlib.import_module("src.utils.logger")

        logger1 = logger_module.get_logger("notification-service-test")
        logger2 = logger_module.get_logger("notification-service-test")

        self.assertIs(logger1, logger2)

    def test_json_formatter_format(self):
        logger_module = importlib.import_module("src.utils.logger")
        formatter = logger_module.JsonFormatter()

        import logging

        record = logging.LogRecord(
            name="test",
            level=logging.INFO,
            pathname=__file__,
            lineno=1,
            msg="hello",
            args=(),
            exc_info=None,
        )

        payload = formatter.format(record)

        self.assertIn('"level": "INFO"', payload)
        self.assertIn('"message": "hello"', payload)
        self.assertIn('"timestamp":', payload)


if __name__ == "__main__":
    unittest.main()
