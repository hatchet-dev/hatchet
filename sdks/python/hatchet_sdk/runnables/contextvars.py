import asyncio
import threading
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


class TaskCounter:
    def __init__(self) -> None:
        self._count = 0
        self._lock = threading.Lock()

    def increment(self) -> int:
        with self._lock:
            self._count += 1
            return self._count

    @property
    def value(self) -> int:
        return self._count


task_count = TaskCounter()
