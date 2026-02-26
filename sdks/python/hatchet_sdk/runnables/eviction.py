from __future__ import annotations

from datetime import timedelta

from pydantic import BaseModel, ConfigDict


class EvictionPolicy(BaseModel):
    """
    Task-scoped eviction parameters for *durable* tasks.

    :ivar ttl: Maximum continuous waiting duration before TTL-eligible eviction.
        Applies to time spent in SDK-instrumented "waiting" states (e.g.
        `ctx.aio_wait_for(...)`, waiting for a workflow run result).
    :ivar allow_capacity_eviction: Whether this task may be evicted under durable-slot pressure.
    :ivar priority: Lower values are evicted first when multiple candidates exist.

    Setting the durable task's eviction params to `None` means the task run is
    never eligible for eviction.

    **Example**
    ```python
    EvictionPolicy(
        ttl=timedelta(minutes=10),
        allow_capacity_eviction=True,
        priority=0,
    )
    ```
    """

    model_config = ConfigDict(frozen=True)

    ttl: timedelta | None

    allow_capacity_eviction: bool = True

    priority: int = 0


# Shared sensible defaults (single source of truth).
# NOTE: When changing these values, update the :param durable_run_eviction: / :param eviction_policy:
# docstrings in workflow.Workflow.durable_task and hatchet.Hatchet.durable_task to match.
DEFAULT_DURABLE_TASK_EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(minutes=15),
    allow_capacity_eviction=True,
    priority=0,
)
