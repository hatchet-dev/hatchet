from __future__ import annotations

import asyncio
import threading
from collections import Counter
from contextvars import ContextVar
from typing import TYPE_CHECKING

from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.utils.typing import JSONSerializableMapping

if TYPE_CHECKING:
    from hatchet_sdk.context.context import Context

ctx_workflow_run_id: ContextVar[str | None] = ContextVar(
    "ctx_workflow_run_id", default=None
)
ctx_action_key: ContextVar[ActionKey | None] = ContextVar(
    "ctx_action_key", default=None
)
ctx_step_run_id: ContextVar[str | None] = ContextVar("ctx_step_run_id", default=None)
ctx_worker_id: ContextVar[str | None] = ContextVar("ctx_worker_id", default=None)
ctx_additional_metadata: ContextVar[JSONSerializableMapping | None] = ContextVar(
    "ctx_additional_metadata", default=None
)
ctx_task_retry_count: ContextVar[int | None] = ContextVar(
    "ctx_task_retry_count", default=0
)


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

ctx_hatchet_context: ContextVar[Context | None] = ContextVar(
    "ctx_hatchet_context", default=None
)
