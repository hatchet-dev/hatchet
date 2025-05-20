import asyncio
from collections import Counter
from contextvars import ContextVar

from hatchet_sdk.runnables.action import ActionKey

ctx_workflow_run_id: ContextVar[str | None] = ContextVar(
    "ctx_workflow_run_id", default=None
)
ctx_action_key: ContextVar[ActionKey | None] = ContextVar(
    "ctx_action_key", default=None
)
ctx_step_run_id: ContextVar[str | None] = ContextVar("ctx_step_run_id", default=None)
ctx_worker_id: ContextVar[str | None] = ContextVar("ctx_worker_id", default=None)

workflow_spawn_indices = Counter[ActionKey]()
spawn_index_lock = asyncio.Lock()
