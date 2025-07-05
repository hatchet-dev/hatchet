import asyncio
import functools
import logging
from collections.abc import Awaitable, Callable
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
from hatchet_sdk.utils.typing import STOP_LOOP, STOP_LOOP_TYPE

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


class LogRecord(BaseModel):
    message: str
    step_run_id: str


class AsyncLogSender:
    def __init__(self, event_client: EventClient):
        self.event_client = event_client
        self.q = asyncio.Queue[LogRecord | STOP_LOOP_TYPE](maxsize=1000)

    async def consume(self) -> None:
        while True:
            record = await self.q.get()

            if record == STOP_LOOP:
                break

            try:
                self.event_client.log(
                    message=record.message, step_run_id=record.step_run_id
                )
            except Exception as e:
                logger.error(f"Error logging: {e}")

    def publish(self, record: LogRecord | STOP_LOOP_TYPE) -> None:
        try:
            self.q.put_nowait(record)
        except asyncio.QueueFull:
            logger.warning("Log queue is full, dropping log message")


class CustomLogHandler(logging.StreamHandler):  # type: ignore[type-arg]
    def __init__(self, log_sender: AsyncLogSender, stream: StringIO):
        super().__init__(stream)

        self.log_sender = log_sender

    def emit(self, record: logging.LogRecord) -> None:
        super().emit(record)

        log_entry = self.format(record)
        step_run_id = ctx_step_run_id.get()

        if not step_run_id:
            return

        self.log_sender.publish(LogRecord(message=log_entry, step_run_id=step_run_id))


def capture_logs(
    logger: logging.Logger, log_sender: AsyncLogSender, func: Callable[P, Awaitable[T]]
) -> Callable[P, Awaitable[T]]:
    @functools.wraps(func)
    async def wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
        log_stream = StringIO()
        custom_handler = CustomLogHandler(log_sender, log_stream)
        custom_handler.setLevel(logging.INFO)

        if not any(h for h in logger.handlers if isinstance(h, CustomLogHandler)):
            logger.addHandler(custom_handler)

        try:
            result = await func(*args, **kwargs)
        finally:
            custom_handler.flush()
            logger.removeHandler(custom_handler)
            log_stream.close()

        return result

    return wrapper
