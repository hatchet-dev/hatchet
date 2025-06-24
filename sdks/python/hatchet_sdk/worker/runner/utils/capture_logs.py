import functools
import logging
from collections.abc import Awaitable, Callable
from concurrent.futures import ThreadPoolExecutor
from io import StringIO
from typing import Literal, ParamSpec, TypeVar

from pydantic import BaseModel

from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_step_run_id,
    ctx_worker_id,
    ctx_workflow_run_id,
)

T = TypeVar("T")
P = ParamSpec("P")


class ContextVarToCopy(BaseModel):
    name: Literal[
        "ctx_workflow_run_id", "ctx_step_run_id", "ctx_action_key", "ctx_worker_id"
    ]
    value: str | None


def copy_context_vars(
    ctx_vars: list[ContextVarToCopy],
    func: Callable[P, T],
    *args: P.args,
    **kwargs: P.kwargs,
) -> T:
    for var in ctx_vars:
        if var.name == "ctx_workflow_run_id":
            ctx_workflow_run_id.set(var.value)
        elif var.name == "ctx_step_run_id":
            ctx_step_run_id.set(var.value)
        elif var.name == "ctx_action_key":
            ctx_action_key.set(var.value)
        elif var.name == "ctx_worker_id":
            ctx_worker_id.set(var.value)
        else:
            raise ValueError(f"Unknown context variable name: {var.name}")

    return func(*args, **kwargs)


class CustomLogHandler(logging.StreamHandler):  # type: ignore[type-arg]
    def __init__(self, event_client: EventClient, stream: StringIO):
        super().__init__(stream)

        self.logger_thread_pool = ThreadPoolExecutor(max_workers=1)
        self.event_client = event_client

    def _log(self, line: str, step_run_id: str | None) -> None:
        if not step_run_id:
            return

        try:
            self.event_client.log(message=line, step_run_id=step_run_id)
        except Exception as e:
            logger.error(f"Error logging: {e}")

    def emit(self, record: logging.LogRecord) -> None:
        super().emit(record)

        log_entry = self.format(record)

        self.logger_thread_pool.submit(self._log, log_entry, ctx_step_run_id.get())


def capture_logs(
    logger: logging.Logger, event_client: "EventClient", func: Callable[P, Awaitable[T]]
) -> Callable[P, Awaitable[T]]:
    @functools.wraps(func)
    async def wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
        log_stream = StringIO()
        custom_handler = CustomLogHandler(event_client, log_stream)
        custom_handler.setLevel(logging.INFO)

        if not any(h for h in logger.handlers if isinstance(h, CustomLogHandler)):
            logger.addHandler(custom_handler)

        try:
            await func(*args, **kwargs)
        finally:
            custom_handler.flush()
            logger.removeHandler(custom_handler)
            log_stream.close()

    return wrapper
