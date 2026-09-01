from enum import Enum

from pydantic import BaseModel, Field

from hatchet_sdk.contracts.v1.workflows_pb2 import Concurrency


class ConcurrencyLimitStrategy(str, Enum):
    CANCEL_IN_PROGRESS = "CANCEL_IN_PROGRESS"
    GROUP_ROUND_ROBIN = "GROUP_ROUND_ROBIN"
    CANCEL_NEWEST = "CANCEL_NEWEST"
    QUEUE_NEWEST = "QUEUE_NEWEST"
    QUEUE_OLDEST = "QUEUE_OLDEST"
    CANCEL_QUEUED_EXCEPT_NEWEST = "CANCEL_QUEUED_EXCEPT_NEWEST"
    CANCEL_QUEUED_EXCEPT_OLDEST = "CANCEL_QUEUED_EXCEPT_OLDEST"


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
    max_runs: int = Field(gt=0)
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


class SharedConcurrency(BaseModel):
    """
    A tenant-scoped concurrency strategy, shared across workflows: every task declaring the
    same name consumes the same concurrency limit, and re-declaring a name updates the
    strategy in place. Declare it anywhere a `ConcurrencyExpression` is accepted; the
    position in the concurrency list is the chain order, so it may come before or after
    workflow-scoped entries. Chains sharing tenant-scoped strategies must order them
    consistently, or registration is rejected.

    Args:
        name (str): Unique (per tenant) name of the strategy.
        expression (str): CEL expression to determine concurrency grouping. (i.e. "input.user_id")
        max_runs (int): Maximum number of concurrent runs, defaults to 1.
        limit_strategy (ConcurrencyLimitStrategy): Strategy for handling limit violations.
    """

    name: str
    expression: str
    max_runs: int = Field(gt=0, default=1)
    limit_strategy: ConcurrencyLimitStrategy = ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS

    def to_proto(self) -> Concurrency:
        return Concurrency(
            name=self.name,
            tenant_scoped=True,
            expression=self.expression,
            max_runs=self.max_runs,
            limit_strategy=self.limit_strategy,
        )
