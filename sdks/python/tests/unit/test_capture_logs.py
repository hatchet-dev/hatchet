from __future__ import annotations

import asyncio
import logging
from types import SimpleNamespace
from typing import cast

from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.runnables.contextvars import ctx_step_run_id, ctx_task_retry_count
from hatchet_sdk.utils.typing import LogLevel
from hatchet_sdk.worker.runner.utils.capture_logs import (
    AsyncLogSender,
    LogForwardingHandler,
    LogRecord,
)


class FakeEventClient:
    def __init__(self) -> None:
        self.client_config = SimpleNamespace(log_queue_size=10)


async def test_log_forwarding_handler_enqueues_correct_record() -> None:
    event_client = FakeEventClient()
    log_sender = AsyncLogSender(cast(EventClient, event_client))

    target_logger = logging.getLogger("capture-log-test")
    previous_level = target_logger.level
    target_logger.setLevel(logging.INFO)

    handler = LogForwardingHandler(log_sender)
    target_logger.addHandler(handler)

    step_token = ctx_step_run_id.set("step-run-id")
    retry_token = ctx_task_retry_count.set(2)

    try:

        def log_from_worker_thread() -> None:
            logging.getLogger("capture-log-test").info("hello from worker thread")

        await asyncio.to_thread(log_from_worker_thread)

        record = log_sender.q.get()
        assert isinstance(record, LogRecord)
        assert record.message == "hello from worker thread"
        assert record.step_run_id == "step-run-id"
        assert record.level == LogLevel.INFO
        assert record.task_retry_count == 2
    finally:
        ctx_step_run_id.reset(step_token)
        ctx_task_retry_count.reset(retry_token)
        target_logger.removeHandler(handler)
        target_logger.setLevel(previous_level)


def test_publish_drops_records_once_queue_is_full() -> None:
    event_client = FakeEventClient()
    log_sender = AsyncLogSender(cast(EventClient, event_client))
    log_queue_size = event_client.client_config.log_queue_size

    for index in range(log_queue_size + 5):
        log_sender.publish(
            LogRecord(
                message=f"message-{index}",
                step_run_id="step-run-id",
                level=LogLevel.INFO,
                task_retry_count=0,
            )
        )

    queued_messages = [
        cast(LogRecord, log_sender.q.get_nowait()).message
        for _ in range(log_sender.q.qsize())
    ]

    assert queued_messages == [f"message-{index}" for index in range(log_queue_size)]


def test_dropped_log_warning_does_not_recurse_when_sdk_logger_is_captured() -> None:
    event_client = FakeEventClient()
    event_client.client_config.log_queue_size = 1
    log_sender = AsyncLogSender(cast(EventClient, event_client))

    sdk_logger = logging.getLogger("hatchet")
    handler = LogForwardingHandler(log_sender)
    sdk_logger.addHandler(handler)
    step_token = ctx_step_run_id.set("step-run-id")

    try:
        for index in range(3):
            sdk_logger.info("message-%d", index)
    finally:
        ctx_step_run_id.reset(step_token)
        sdk_logger.removeHandler(handler)

    assert log_sender.q.qsize() == 1
