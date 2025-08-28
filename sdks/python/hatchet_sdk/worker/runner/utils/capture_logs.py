import asyncio
import copy
import functools
import logging
from collections.abc import Awaitable, Callable
from datetime import UTC, datetime, timedelta
from io import StringIO
from typing import Literal, ParamSpec, TypeVar
from uuid import UUID, uuid4

import grpc
from pydantic import BaseModel, Field

from hatchet_sdk.clients.events import BulkLogPushRequest, EventClient
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_additional_metadata,
    ctx_step_run_id,
    ctx_worker_id,
    ctx_workflow_run_id,
)
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


class ContextVarToCopyDict(BaseModel):
    name: Literal["ctx_additional_metadata"]
    value: JSONSerializableMapping | None


class ContextVarToCopy(BaseModel):
    var: ContextVarToCopyStr | ContextVarToCopyDict = Field(discriminator="name")


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
        elif var.var.name == "ctx_action_key":
            ctx_action_key.set(var.var.value)
        elif var.var.name == "ctx_worker_id":
            ctx_worker_id.set(var.var.value)
        elif var.var.name == "ctx_additional_metadata":
            ctx_additional_metadata.set(var.var.value or {})
        else:
            raise ValueError(f"Unknown context variable name: {var.var.name}")

    return func(*args, **kwargs)


class LogRecord(BaseModel):
    id: UUID = Field(default_factory=uuid4)
    message: str
    task_run_external_id: str
    level: LogLevel
    timestamp: datetime

    def __hash__(self) -> int:
        return hash(self.id)


class LogBuffer(BaseModel):
    records: set[LogRecord] = Field(default_factory=set)

    def should_clear(self, config: ClientConfig) -> bool:
        if not self.records:
            return False

        if len(self.records) >= config.log_buffer_size:
            return True

        oldest_ts = min(r.timestamp for r in self.records)

        return datetime.now(UTC) - oldest_ts > timedelta(
            seconds=config.log_flush_interval_seconds
        )


class AsyncLogSender:
    def __init__(self, event_client: EventClient):
        self.event_client = event_client
        self.q = asyncio.Queue[LogRecord | STOP_LOOP_TYPE](
            maxsize=event_client.client_config.log_queue_size
        )
        self.buffer = LogBuffer()

        ## flag to see if the engine supports bulk log push
        ## defaults to true, but reset to false if we get an
        ## `UNIMPLEMENTED` or `NOT_FOUND` error back
        self.bulk_log_push_enabled = True

    async def flush(self) -> None:
        ## IMPORTANT: Need a deepcopy here to act as a snapshot of the
        ## current buffer, since new logs might be added while we're processing
        ## the current buffer
        records = copy.deepcopy(self.buffer.records)

        requests = [
            BulkLogPushRequest(
                message=r.message,
                task_run_external_id=r.task_run_external_id,
                level=r.level,
            )
            for r in records
        ]

        try:
            await asyncio.to_thread(self.event_client.bulk_log, requests)
            self.buffer.records -= records
        except grpc.RpcError as e:
            if e.code() in [grpc.StatusCode.UNIMPLEMENTED, grpc.StatusCode.NOT_FOUND]:
                self.bulk_log_push_enabled = False
                for record in records:
                    await self.push_single(record)
                    self.buffer.records.remove(record)

        except Exception:
            logger.exception("failed to send log to Hatchet")

    async def push_single(self, record: LogRecord) -> None:
        try:
            await asyncio.to_thread(
                self.event_client.log,
                message=record.message,
                step_run_id=record.task_run_external_id,
                level=record.level,
            )
        except Exception:
            logger.exception("failed to send log to Hatchet")

    async def get_log(self) -> LogRecord | STOP_LOOP_TYPE | None:
        try:
            async with asyncio.timeout(
                self.event_client.client_config.log_flush_interval_seconds
            ):
                return await self.q.get()
        except TimeoutError:
            return None

    async def consume(self) -> None:
        while True:
            record = await self.get_log()

            if record == STOP_LOOP:
                await self.flush()
                break

            if self.bulk_log_push_enabled:
                if record:
                    self.buffer.records.add(record)

                if self.buffer.should_clear(self.event_client.client_config):
                    await self.flush()
            else:
                if record:
                    await self.push_single(record)

    def publish(self, record: LogRecord | STOP_LOOP_TYPE) -> None:
        try:
            self.q.put_nowait(record)
        except asyncio.QueueFull:
            logger.warning("log queue is full, dropping log message")


class LogForwardingHandler(logging.StreamHandler):  # type: ignore[type-arg]
    def __init__(self, log_sender: AsyncLogSender, stream: StringIO):
        super().__init__(stream)

        self.log_sender = log_sender

    def emit(self, record: logging.LogRecord) -> None:
        super().emit(record)

        log_entry = self.format(record)
        step_run_id = ctx_step_run_id.get()

        if not step_run_id:
            return

        self.log_sender.publish(
            LogRecord(
                message=log_entry,
                task_run_external_id=step_run_id,
                level=LogLevel.from_levelname(record.levelname),
                timestamp=datetime.now(UTC),
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
