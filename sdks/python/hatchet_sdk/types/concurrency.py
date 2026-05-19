from enum import Enum

from pydantic import BaseModel

from hatchet_sdk.contracts.v1.workflows_pb2 import Concurrency


class ConcurrencyLimitStrategy(str, Enum):
    CANCEL_IN_PROGRESS = "CANCEL_IN_PROGRESS"
    GROUP_ROUND_ROBIN = "GROUP_ROUND_ROBIN"
    CANCEL_NEWEST = "CANCEL_NEWEST"


class ConcurrencyExpression(BaseModel):
    """
    Defines concurrency limits for a workflow using a CEL expression.
    Args:
        expression (str): CEL expression to determine concurrency grouping. (i.e. "input.user_id")
        max_runs (int): Maximum number of concurrent workflow runs.
        limit_strategy (ConcurrencyLimitStrategy): Strategy for handling limit violations.
    Example:
        ConcurrencyExpression("input.user_id", 5, ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS)
    """

    expression: str
    max_runs: int
    limit_strategy: ConcurrencyLimitStrategy

    def to_proto(self) -> Concurrency:
        return Concurrency(
            expression=self.expression,
            max_runs=self.max_runs,
            limit_strategy=self.limit_strategy,
        )

    @staticmethod
    def from_int(max_runs: int) -> "ConcurrencyExpression":
        return ConcurrencyExpression(
            expression="'constant'",
            max_runs=max_runs,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        )
