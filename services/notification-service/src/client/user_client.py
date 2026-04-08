import os
import sys

import grpc

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../shared/python/v1"))

from user import user_pb2, user_pb2_grpc  # noqa: E402
from src.utils.logger import logger  # noqa: E402


class UserClient:
    def __init__(self, addr: str):
        self._channel = grpc.insecure_channel(addr)
        self._stub = user_pb2_grpc.UserServiceStub(self._channel)

    def get_profile(self, user_id: str) -> user_pb2.User | None:
        try:
            response = self._stub.GetProfile(user_pb2.GetProfileRequest(id=user_id))
            return response.user
        except grpc.RpcError as e:
            logger.error(
                f"Failed to get user: {e.details()}",
                extra={
                    "user_id": user_id,
                    "grpc_code": e.code().name,
                },
            )
            return None

    def close(self):
        self._channel.close()
