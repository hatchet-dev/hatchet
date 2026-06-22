import contextlib
import functools
import logging
import multiprocessing
import multiprocessing.context
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from io import StringIO
from typing import Any, Literal, ParamSpec, TypeVar

from pydantic import BaseModel, Field

from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_additional_metadata,
    ctx_hatchet_context,
    ctx_hatchet_span_attributes,
    ctx_step_run_id,
    ctx_task_retry_count,
    ctx_worker_id,
    ctx_workflow_run_id,
)

try:
    from opentelemetry import context as otel_context

    _HAS_OTEL = True
except ImportError:
    _HAS_OTEL = False
from hatchet_sdk.utils.typing import (
    STOP_LOOP,
    STOP_LOOP_TYPE,
    JSONSerializableMapping,
    LogLevel,
)

T = TypeVar("T")
P = ParamSpec("P")


class ContextVarToCopyStr(BaseModel):
    name: Literal[
        "ctx_workflow_run_id",
        "ctx_step_run_id",
        "ctx_action_key",
        "ctx_worker_id",
    ]
    value: str | None


class ContextVarToCopyInt(BaseModel):
    name: Literal["ctx_task_retry_count"]
    value: int | None


class ContextVarToCopyDict(BaseModel):
    name: Literal["ctx_additional_metadata"]
    value: JSONSerializableMapping | None


class ContextVarToCopyHatchetContext(BaseModel):
    name: Literal["ctx_hatchet_context"]
    value: Any


class ContextVarToCopySpanAttributes(BaseModel):
    name: Literal["ctx_hatchet_span_attributes"]
    value: dict[str, str | int] | None


class ContextVarToCopyOtelContext(BaseModel):
    name: Literal["ctx_otel_context"]
    value: Any  # opentelemetry.context.Context (optional dependency)


class ContextVarToCopy(BaseModel):
    var: (
        ContextVarToCopyStr
        | ContextVarToCopyDict
        | ContextVarToCopyInt
        | ContextVarToCopyHatchetContext
        | ContextVarToCopySpanAttributes
        | ContextVarToCopyOtelContext
    ) = Field(discriminator="name")


def copy_context_vars(
    ctx_vars: list[ContextVarToCopy],
    func: Callable[P, T],
    *args: P.args,
    **kwargs: P.kwargs,
) -> T:
    for var in ctx_vars:
        if var.var.name == "ctx_workflow_run_id":
            ctx_workflow_run_id.set(var.var.value)
        elif var.var.name == "ctx_step_run_id":
            ctx_step_run_id.set(var.var.value)
        elif var.var.name == "ctx_task_retry_count":
            ctx_task_retry_count.set(var.var.value)
        elif var.var.name == "ctx_action_key":
            ctx_action_key.set(var.var.value)
        elif var.var.name == "ctx_worker_id":
            ctx_worker_id.set(var.var.value)
        elif var.var.name == "ctx_additional_metadata":
            ctx_additional_metadata.set(var.var.value or {})
        elif var.var.name == "ctx_hatchet_context":
            ctx_hatchet_context.set(var.var.value)
        elif var.var.name == "ctx_hatchet_span_attributes":
            ctx_hatchet_span_attributes.set(var.var.value)
        elif var.var.name == "ctx_otel_context":
            if _HAS_OTEL and var.var.value is not None:
                otel_context.attach(var.var.value)
        else:
            raise ValueError(f"Unknown context variable name: {var.var.name}")

    return func(*args, **kwargs)


@dataclass
class LogRecord:
    message: str
    step_run_id: str
    level: LogLevel
    task_retry_count: int


class AsyncLogSender:
    def __init__(self, event_client: EventClient):
        self._config = event_client.client_config
        self._ctx = multiprocessing.get_context("spawn")
        self.q: multiprocessing.Queue[LogRecord | STOP_LOOP_TYPE] = self._ctx.Queue(
            maxsize=self._config.log_queue_size
        )
        self._proc: multiprocessing.context.SpawnProcess | None = None

    def publish(self, record: LogRecord | STOP_LOOP_TYPE) -> None:
        self.q.put_nowait(record)

    def consume(
        self,
        q: multiprocessing.Queue[LogRecord | STOP_LOOP_TYPE],
        config: ClientConfig,
    ) -> None:
        import queue

        from hatchet_sdk.clients.events import EventClient

        client = EventClient(config)

        while True:
            try:
                record = q.get(timeout=1)
            except queue.Empty:
                # Parent may have died without sending STOP_LOOP; keep polling
                # so we don't block forever on a dead queue.
                continue
            if record == STOP_LOOP:
                break
            try:
                client.log(
                    message=record.message,
                    step_run_id=record.step_run_id,
                    level=record.level,
                    task_retry_count=record.task_retry_count,
                )
            except Exception:
                logger.exception("failed to send log to Hatchet")

    def start(self) -> None:
        proc = self._ctx.Process(
            target=self.consume,
            args=(self.q, self._config),
            daemon=True,
        )
        self._proc = proc
        proc.start()

    def stop(self, timeout: float = 5.0) -> None:
        proc = self._proc
        if proc is None:
            return
        if proc.is_alive():
            with contextlib.suppress(Exception):
                self.q.put_nowait(STOP_LOOP)
            proc.join(timeout)
            if proc.is_alive():
                proc.terminate()
                proc.join(timeout)
        self.q.close()
        self.q.join_thread()
        self._proc = None


class LogForwardingHandler(logging.StreamHandler):  # type: ignore[type-arg]
    def __init__(self, log_sender: AsyncLogSender, stream: StringIO):
        super().__init__(stream)

        self.log_sender = log_sender

    def emit(self, record: logging.LogRecord) -> None:
        super().emit(record)

        log_entry = self.format(record)
        step_run_id = ctx_step_run_id.get()
        task_retry_count = ctx_task_retry_count.get()

        if not step_run_id:
            return

        self.log_sender.publish(
            LogRecord(
                message=log_entry,
                step_run_id=step_run_id,
                level=LogLevel.from_levelname(record.levelname),
                task_retry_count=task_retry_count or 0,
            )
        )


def capture_logs(
    logger: logging.Logger, log_sender: AsyncLogSender, func: Callable[P, Awaitable[T]]
) -> Callable[P, Awaitable[T]]:
    @functools.wraps(func)
    async def wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
        log_stream = StringIO()
        log_forwarder = LogForwardingHandler(log_sender, log_stream)
        log_forwarder.setLevel(logger.level)

        if logger.handlers:
            for handler in logger.handlers:
                if handler.formatter:
                    log_forwarder.setFormatter(handler.formatter)
                    break

            for handler in logger.handlers:
                for filter_obj in handler.filters:
                    log_forwarder.addFilter(filter_obj)

        if not any(h for h in logger.handlers if isinstance(h, LogForwardingHandler)):
            logger.addHandler(log_forwarder)

        try:
            result = await func(*args, **kwargs)
        finally:
            log_forwarder.flush()
            logger.removeHandler(log_forwarder)
            log_stream.close()

        return result

    return wrapper
