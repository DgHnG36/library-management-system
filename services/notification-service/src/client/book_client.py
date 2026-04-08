import os
import sys

import grpc

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../shared/python/v1"))

from book import book_pb2, book_pb2_grpc  # noqa: E402
from src.utils.logger import logger  # noqa: E402


class BookClient:
    def __init__(self, addr: str):
        self._channel = grpc.insecure_channel(addr)
        self._stub = book_pb2_grpc.BookServiceStub(self._channel)

    def get_book(self, book_id: str) -> book_pb2.Book | None:
        try:
            response = self._stub.GetBook(book_pb2.GetBookRequest(id=book_id))
            return response.book
        except grpc.RpcError as e:
            logger.error(
                f"Failed to get book: {e.details()}",
                extra={
                    "book_id": book_id,
                    "grpc_code": e.code().name,
                },
            )
            return None

    def close(self):
        self._channel.close()
