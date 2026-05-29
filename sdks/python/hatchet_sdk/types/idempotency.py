from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk.contracts.v1.workflows_pb2 import (
    IdempotencyConfig as IdempotencyConfigProto,
)


class IdempotencyConfig(BaseModel):
    key_expression: str
    ttl: timedelta | None

    def to_proto(self) -> IdempotencyConfigProto:
        ttl_ms = int(self.ttl.total_seconds() * 1000) if self.ttl is not None else None
        return IdempotencyConfigProto(
            expression=self.key_expression,
            ttl_ms=ttl_ms,
        )
