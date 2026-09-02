from enum import Enum

from pydantic import BaseModel, field_validator

from hatchet_sdk.contracts.v1.workflows_pb2 import Concurrency


class ConcurrencyLimitStrategy(str, Enum):
    CANCEL_IN_PROGRESS = "CANCEL_IN_PROGRESS"
    GROUP_ROUND_ROBIN = "GROUP_ROUND_ROBIN"
    CANCEL_NEWEST = "CANCEL_NEWEST"
    QUEUE_NEWEST = "QUEUE_NEWEST"
    QUEUE_OLDEST = "QUEUE_OLDEST"
    CANCEL_QUEUED_EXCEPT_NEWEST = "CANCEL_QUEUED_EXCEPT_NEWEST"
    CANCEL_QUEUED_EXCEPT_OLDEST = "CANCEL_QUEUED_EXCEPT_OLDEST"


def _validate_max_runs(v: int | str) -> int | str:
    if isinstance(v, int) and v <= 0:
        raise ValueError("max_runs must be greater than 0")
    if isinstance(v, str) and not v.strip():
        raise ValueError("max_runs expression must be non-empty")
    return v


def _max_runs_to_proto(v: int | str) -> tuple[int, str | None]:
    """Split the union onto the proto's static/expression field pair. When a CEL
    expression is used, the static field carries the default of 1, which only governs
    slots created before the expression existed (each new task's slot carries its own
    evaluated value)."""
    if isinstance(v, str):
        return 1, v

    return v, None


class ConcurrencyExpression(BaseModel):
    """
    Defines concurrency limits for a workflow using a CEL expression.
    Args:
        expression (str): CEL expression to determine concurrency grouping. (i.e. "input.user_id")
        max_runs (int | str): Maximum number of concurrent runs, either a fixed number or
            a CEL expression over task input computing the max runs for that task's
            concurrency group (i.e. "input.tier == 'premium' ? 10 : 1"). With an
            expression, a group's effective limit is the value from its most recently
            created task.
        limit_strategy (ConcurrencyLimitStrategy): Strategy for handling limit violations.
    Example:
        ConcurrencyExpression("input.user_id", 5, ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS)
    """

    expression: str
    max_runs: int | str
    limit_strategy: ConcurrencyLimitStrategy

    @field_validator("max_runs")
    @classmethod
    def validate_max_runs(cls, v: int | str) -> int | str:
        return _validate_max_runs(v)

    def to_proto(self) -> Concurrency:
        max_runs, max_runs_expression = _max_runs_to_proto(self.max_runs)

        return Concurrency(
            expression=self.expression,
            max_runs=max_runs,
            limit_strategy=self.limit_strategy,
            max_runs_expression=max_runs_expression,
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
        max_runs (int | str): Maximum number of concurrent runs, either a fixed number
            (defaults to 1) or a CEL expression over task input computing the max runs for
            that task's concurrency group. With an expression, a group's effective limit
            is the value from its most recently created task.
        limit_strategy (ConcurrencyLimitStrategy): Strategy for handling limit violations.
    """

    name: str
    expression: str
    max_runs: int | str = 1
    limit_strategy: ConcurrencyLimitStrategy = (
        ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS
    )

    @field_validator("max_runs")
    @classmethod
    def validate_max_runs(cls, v: int | str) -> int | str:
        return _validate_max_runs(v)

    def to_proto(self) -> Concurrency:
        max_runs, max_runs_expression = _max_runs_to_proto(self.max_runs)

        return Concurrency(
            name=self.name,
            is_tenant_scoped=True,
            expression=self.expression,
            max_runs=max_runs,
            limit_strategy=self.limit_strategy,
            max_runs_expression=max_runs_expression,
        )
