from abc import ABC, abstractmethod
from datetime import timedelta
from enum import Enum

from pydantic import BaseModel

from hatchet_sdk.contracts.v1.workflows_pb2 import (
    IdempotencyConfig as IdempotencyConfigProto,
)
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    IdempotencyMethod as IdempotencyMethodProto,
)
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto


class IdempotencyMethod(Enum):
    TTL = "TTL"
    STATUS = "STATUS"


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
            method=convert_python_enum_to_proto(  # type: ignore[arg-type]
                IdempotencyMethod.TTL, IdempotencyMethodProto
            ),
        )


class StatusBasedIdempotencyConfig(BaseIdemotencyConfig):
    fallback_ttl: timedelta

    def to_proto(self) -> IdempotencyConfigProto:
        return IdempotencyConfigProto(
            expression=self.key_expression,
            ttl_ms=int(self.fallback_ttl.total_seconds() * 1000),
            method=convert_python_enum_to_proto(  # type: ignore[arg-type]
                IdempotencyMethod.STATUS, IdempotencyMethodProto
            ),
        )
