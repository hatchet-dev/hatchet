from enum import Enum

from pydantic import BaseModel, field_validator, model_validator

from hatchet_sdk.contracts.v1.workflows_pb2 import Concurrency


class ConcurrencyLimitStrategy(str, Enum):
    CANCEL_IN_PROGRESS = "CANCEL_IN_PROGRESS"
    GROUP_ROUND_ROBIN = "GROUP_ROUND_ROBIN"
    CANCEL_NEWEST = "CANCEL_NEWEST"
    CANCEL_QUEUED_EXCEPT_NEWEST = "CANCEL_QUEUED_EXCEPT_NEWEST"
    CANCEL_QUEUED_EXCEPT_OLDEST = "CANCEL_QUEUED_EXCEPT_OLDEST"


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
        name (str | None): Unique (per tenant) strategy name. Required when
            `is_tenant_scoped` is set.
        is_tenant_scoped (bool): When True, the entry defines (or updates in place) a
            tenant-scoped strategy shared across workflows, keyed by `name`: every task
            declaring the same name consumes the same concurrency limit. The position in
            the concurrency list is the chain order, and chains sharing tenant-scoped
            strategies must order them consistently.

    Example:
        ConcurrencyExpression("input.user_id", 5, ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS)

    Example (tenant-scoped, shared across workflows):
        ConcurrencyExpression(
            expression="input.group",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
            name="tenant-wide-limit",
            is_tenant_scoped=True,
        )
    """

    expression: str
    max_runs: int | str
    limit_strategy: ConcurrencyLimitStrategy
    name: str | None = None
    is_tenant_scoped: bool = False

    @field_validator("max_runs")
    @classmethod
    def validate_max_runs(cls, v: int | str) -> int | str:
        if isinstance(v, int) and v <= 0:
            raise ValueError("max_runs must be greater than 0")
        if isinstance(v, str) and not v.strip():
            raise ValueError("max_runs expression must be non-empty")
        return v

    @model_validator(mode="after")
    def validate_tenant_scoped_name(self) -> "ConcurrencyExpression":
        if self.is_tenant_scoped and not (self.name or "").strip():
            raise ValueError("a name is required for tenant-scoped concurrency")
        return self

    def to_proto(self) -> Concurrency:
        # a string max_runs is a CEL expression; the static field then carries the
        # default of 1, which only governs slots created before the expression existed
        # (each new task's slot carries its own evaluated value)
        if isinstance(self.max_runs, str):
            max_runs, max_runs_expression = 1, self.max_runs
        else:
            max_runs, max_runs_expression = self.max_runs, None

        proto = Concurrency(
            expression=self.expression,
            max_runs=max_runs,
            limit_strategy=self.limit_strategy,
            name=self.name,
            max_runs_expression=max_runs_expression,
        )

        # leave the optional field unset for ordinary entries
        if self.is_tenant_scoped:
            proto.is_tenant_scoped = True

        return proto

    @staticmethod
    def from_int(max_runs: int) -> "ConcurrencyExpression":
        return ConcurrencyExpression(
            expression="'constant'",
            max_runs=max_runs,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        )
