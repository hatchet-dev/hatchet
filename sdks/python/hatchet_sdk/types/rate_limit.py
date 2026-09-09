from typing import Literal

from pydantic import BaseModel, model_validator

from hatchet_sdk.contracts.v1.workflows_pb2 import CreateTaskRateLimit
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    RateLimitDuration as RateLimitDurationProto,
)
from hatchet_sdk.utils.proto_enums import convert_python_literal_to_proto

RateLimitDuration = Literal[
    "SECOND",
    "MINUTE",
    "HOUR",
    "DAY",
    "WEEK",
    "MONTH",
    "YEAR",
]


class RateLimit(BaseModel):
    """
    Represents a rate limit configuration for a step in a workflow.

    This class allows for both static and dynamic rate limiting based on various parameters.
    It supports both simple integer values and Common Expression Language (CEL) expressions
    for dynamic evaluation.

    Attributes:
        static_key (str, optional): A static key for rate limiting.
        dynamic_key (str, optional): A CEL expression for dynamic key evaluation.
        units (int or str, default=1): The number of units or a CEL expression for dynamic unit calculation.
        limit (int or str, optional): The rate limit value or a CEL expression for dynamic limit calculation.
        duration (str, default="MINUTE"): The window duration of the rate limit.

    Usage:
        1. Static rate limit:
           rate_limit = RateLimit(static_key="external-api", units=100)
           > NOTE: if you want to use a static key, you must first put the rate limit: hatchet.admin.put_rate_limit("external-api", 200, "SECOND")

        2. Dynamic rate limit with CEL expressions:
           rate_limit = RateLimit(
               dynamic_key="input.user_id",
               units="input.units",
               limit="input.limit * input.user_tier"
           )

    Note:
        - Either static_key or dynamic_key must be set, but not both.
        - When using dynamic_key, limit must also be set.
        - CEL expressions are validated upon instantiation.

    Raises:
        ValueError: If invalid combinations of attributes are provided or if CEL expressions are invalid.
    """

    static_key: str | None = None
    dynamic_key: str | None = None
    units: str | int = 1
    limit: int | str | None = None
    duration: RateLimitDuration = "MINUTE"

    @model_validator(mode="after")
    def validate_rate_limit(self) -> "RateLimit":
        if self.dynamic_key and self.static_key:
            raise ValueError("Cannot have both static key and dynamic key set")

        if self.limit and not isinstance(self.limit, int | str):
            raise ValueError(f"Invalid limit value: {self.limit}")

        if self.dynamic_key and not self.limit:
            raise ValueError("CEL based keys requires limit to be set")

        if self.limit is None:
            self.limit = -1

        return self

    def to_proto(self) -> CreateTaskRateLimit:
        key = self.static_key
        key_expression = self.dynamic_key

        key = self.static_key or self.dynamic_key

        units = self.units if isinstance(self.units, int) else None
        units_expression = None if isinstance(self.units, int) else self.units

        limit_expression = None if not self.limit else str(self.limit)

        return CreateTaskRateLimit(
            key=key,
            key_expr=key_expression,
            units=units,
            units_expr=units_expression,
            limit_values_expr=limit_expression,
            duration=convert_python_literal_to_proto(
                self.duration, RateLimitDurationProto
            ),
        )
