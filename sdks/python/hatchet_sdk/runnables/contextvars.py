import asyncio
from collections import Counter
from contextvars import ContextVar

ctx_workflow_run_id: ContextVar[str | None] = ContextVar(
    "ctx_workflow_run_id", default=None
)
ctx_step_run_id: ContextVar[str | None] = ContextVar("ctx_step_run_id", default=None)
ctx_worker_id: ContextVar[str | None] = ContextVar("ctx_worker_id", default=None)

workflow_spawn_indices = Counter[str]()
spawn_index_lock = asyncio.Lock()
