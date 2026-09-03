from typing import Literal

from pydantic import BaseModel, Field

from hatchet_sdk.contracts.v1.workflows_pb2 import Concurrency
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    ConcurrencyLimitStrategy as ConcurrencyLimitStrategyProto,
)
from hatchet_sdk.utils.proto_enums import convert_python_literal_to_proto


class ConcurrencyStrategy(BaseModel):
    """
    Defines concurrency limits for a workflow using a CEL expression.
    Args:
        expression (str): CEL expression to determine concurrency grouping. (i.e. "input.user_id")
        max_runs (int): Maximum number of concurrent workflow runs.
        strategy (Literal["CANCEL_IN_PROGRESS", "GROUP_ROUND_ROBIN", "CANCEL_NEWEST", "CANCEL_QUEUED_EXCEPT_NEWEST", "CANCEL_QUEUED_EXCEPT_OLDEST"]): Strategy for handling limit violations.
    Example:
        ConcurrencyStrategy("input.user_id", 5, "CANCEL_IN_PROGRESS")
    """

    expression: str
    max_runs: int = Field(gt=0)
    strategy: Literal[
        "CANCEL_IN_PROGRESS",
        "GROUP_ROUND_ROBIN",
        "CANCEL_NEWEST",
        "CANCEL_QUEUED_EXCEPT_NEWEST",
        "CANCEL_QUEUED_EXCEPT_OLDEST",
    ]

    def to_proto(self) -> Concurrency:
        return Concurrency(
            expression=self.expression,
            max_runs=self.max_runs,
            limit_strategy=convert_python_literal_to_proto(
                self.strategy, ConcurrencyLimitStrategyProto
            ),
        )

    @staticmethod
    def from_int(max_runs: int) -> "ConcurrencyStrategy":
        return ConcurrencyStrategy(
            expression="'constant'",
            max_runs=max_runs,
            strategy="GROUP_ROUND_ROBIN",
        )
