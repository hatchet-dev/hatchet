from __future__ import annotations

from datetime import timedelta

from pydantic import BaseModel, ConfigDict


class EvictionPolicy(BaseModel):
    """
    Task-scoped eviction parameters for *durable* tasks.

    Notes:
    - Setting the durable task's eviction params to `None` means the task run is
      never eligible for eviction.
    - `ttl` applies to time spent in SDK-instrumented "waiting" states (e.g.
      `ctx.aio_wait_for(...)`, waiting for a workflow run result).
    """

    model_config = ConfigDict(frozen=True)

    ttl: timedelta | None
    """Maximum continuous waiting duration before TTL-eligible eviction."""

    allow_capacity_eviction: bool = True
    """Whether this task may be evicted under durable-slot pressure."""

    priority: int = 0
    """Lower values are evicted first when multiple candidates exist."""


# Shared sensible defaults (single source of truth).
# NOTE: When changing these values, update the :param durable_run_eviction: / :param eviction:
# docstrings in workflow.Workflow.durable_task and hatchet.Hatchet.durable_task to match.
DEFAULT_DURABLE_TASK_EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(minutes=15),
    allow_capacity_eviction=True,
    priority=0,
)
