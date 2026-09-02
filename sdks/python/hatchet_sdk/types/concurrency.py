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
        max_runs_expression (str | None): CEL expression over task input computing the max
            runs for that task's concurrency group (i.e. "input.tier == 'premium' ? 10 : 1").
            A group's effective limit is the value from its most recently created task.
            Overrides max_runs per group.
    Example:
        ConcurrencyExpression("input.user_id", 5, ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS)
    """

    expression: str
    max_runs: int = Field(gt=0)
    limit_strategy: ConcurrencyLimitStrategy
    max_runs_expression: str | None = None

    def to_proto(self) -> Concurrency:
        return Concurrency(
            expression=self.expression,
            max_runs=self.max_runs,
            limit_strategy=self.limit_strategy,
            max_runs_expression=self.max_runs_expression,
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
        max_runs_expression (str | None): CEL expression over task input computing the max
            runs for that task's concurrency group. A group's effective limit is the value
            from its most recently created task. Overrides max_runs per group.
    """

    name: str
    expression: str
    max_runs: int = Field(gt=0, default=1)
    limit_strategy: ConcurrencyLimitStrategy = (
        ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS
    )
    max_runs_expression: str | None = None

    def to_proto(self) -> Concurrency:
        return Concurrency(
            name=self.name,
            is_tenant_scoped=True,
            expression=self.expression,
            max_runs=self.max_runs,
            limit_strategy=self.limit_strategy,
            max_runs_expression=self.max_runs_expression,
        )
