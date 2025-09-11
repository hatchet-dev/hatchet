from dataclasses import dataclass
from typing import Union

from celpy import CELEvalError, Environment

from hatchet_sdk.contracts.workflows_pb2 import CreateStepRateLimit


def validate_cel_expression(expr: str) -> bool:
    env = Environment()
    try:
        env.compile(expr)
        return True
    except CELEvalError:
        return False


class RateLimitDuration:
    SECOND = "SECOND"
    MINUTE = "MINUTE"
    HOUR = "HOUR"
    DAY = "DAY"
    WEEK = "WEEK"
    MONTH = "MONTH"
    YEAR = "YEAR"


@dataclass
class RateLimit:
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
        duration (str, default=RateLimitDuration.MINUTE): The window duration of the rate limit.
        key (str, optional): Deprecated. Use static_key instead.

    Usage:
        1. Static rate limit:
           rate_limit = RateLimit(static_key="external-api", units=100)
           > NOTE: if you want to use a static key, you must first put the rate limit: hatchet.admin.put_rate_limit("external-api", 200, RateLimitDuration.SECOND)

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
        DeprecationWarning: If the deprecated 'key' attribute is used.
    """

    key: Union[str, None] = None
    static_key: Union[str, None] = None
    dynamic_key: Union[str, None] = None
    units: Union[int, str] = 1
    limit: Union[int, str, None] = None
    duration: RateLimitDuration = RateLimitDuration.MINUTE

    _req: CreateStepRateLimit = None

    def __post_init__(self):
        # juggle the key and key_expr fields
        key = self.static_key
        key_expression = self.dynamic_key

        if self.key is not None:
            DeprecationWarning(
                "key is deprecated and will be removed in a future release, please use static_key instead"
            )
            key = self.key

        if key_expression is not None:
            if key is not None:
                raise ValueError("Cannot have both static key and dynamic key set")

            key = key_expression
            if not validate_cel_expression(key_expression):
                raise ValueError(f"Invalid CEL expression: {key_expression}")

        # juggle the units and units_expr fields
        units = None
        units_expression = None
        if isinstance(self.units, int):
            units = self.units
        else:
            if not validate_cel_expression(self.units):
                raise ValueError(f"Invalid CEL expression: {self.units}")
            units_expression = self.units

        # juggle the limit and limit_expr fields
        limit_expression = None

        if self.limit:
            if isinstance(self.limit, int):
                limit_expression = f"{self.limit}"
            else:
                if not validate_cel_expression(self.limit):
                    raise ValueError(f"Invalid CEL expression: {self.limit}")
                limit_expression = self.limit

        if key_expression is not None and limit_expression is None:
            raise ValueError("CEL based keys requires limit to be set")

        self._req = CreateStepRateLimit(
            key=key,
            key_expr=key_expression,
            units=units,
            units_expr=units_expression,
            limit_values_expr=limit_expression,
            duration=self.duration,
        )
