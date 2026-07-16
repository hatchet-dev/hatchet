from abc import ABC, abstractmethod
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk.contracts.v1.workflows_pb2 import (
    IdempotencyConfig as IdempotencyConfigProto,
)


class BaseIdemotencyConfig(BaseModel, ABC):
    key_expression: str

    @abstractmethod
    def to_proto(self) -> IdempotencyConfigProto: ...


class TTLBasedIdempotencyConfig(BaseIdemotencyConfig):
    ttl: timedelta

    def to_proto(self) -> IdempotencyConfigProto:
        return IdempotencyConfigProto(
            expression=self.key_expression,
            ttl_ms=int(self.ttl.total_seconds() * 1000),
        )
