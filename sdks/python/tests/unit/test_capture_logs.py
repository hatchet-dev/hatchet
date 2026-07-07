from __future__ import annotations

import asyncio
import logging
from io import StringIO
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

    handler = LogForwardingHandler(log_sender, StringIO())
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
