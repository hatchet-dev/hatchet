from __future__ import annotations

import asyncio
import logging
from contextlib import suppress
from io import StringIO
from types import SimpleNamespace
from typing import Any, cast

from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.runnables.contextvars import ctx_step_run_id, ctx_task_retry_count
from hatchet_sdk.utils.typing import STOP_LOOP, LogLevel
from hatchet_sdk.worker.runner.utils.capture_logs import (
    AsyncLogSender,
    LogForwardingHandler,
)


class FakeEventClient:
    def __init__(self, loop: asyncio.AbstractEventLoop) -> None:
        self.client_config = SimpleNamespace(log_queue_size=10)
        self.loop = loop
        self.logged = asyncio.Event()
        self.records: list[dict[str, Any]] = []

    def log(
        self,
        message: str,
        step_run_id: str,
        level: LogLevel | None = None,
        task_retry_count: int | None = None,
    ) -> None:
        self.records.append(
            {
                "message": message,
                "step_run_id": step_run_id,
                "level": level.value if level else None,
                "task_retry_count": task_retry_count,
            }
        )
        self.loop.call_soon_threadsafe(self.logged.set)


async def test_log_forwarding_from_to_thread_uses_sender_loop() -> None:
    loop = asyncio.get_running_loop()
    previous_debug = loop.get_debug()
    loop.set_debug(True)

    event_client = FakeEventClient(loop)
    log_sender = AsyncLogSender(cast(EventClient, event_client))
    consume_task = asyncio.create_task(log_sender.consume())
    await asyncio.sleep(0.01)

    handler = LogForwardingHandler(log_sender, StringIO())
    root_logger = logging.getLogger()
    previous_level = root_logger.level
    root_logger.setLevel(logging.INFO)
    root_logger.addHandler(handler)

    step_token = ctx_step_run_id.set("step-run-id")
    retry_token = ctx_task_retry_count.set(2)
    log_sent = False

    try:

        def log_from_worker_thread() -> None:
            logging.getLogger("capture-log-test").info("hello from worker thread")

        await asyncio.wait_for(asyncio.to_thread(log_from_worker_thread), timeout=1)
        await asyncio.wait_for(event_client.logged.wait(), timeout=1)
        log_sent = True

        assert event_client.records == [
            {
                "message": "hello from worker thread",
                "step_run_id": "step-run-id",
                "level": "INFO",
                "task_retry_count": 2,
            }
        ]
    finally:
        ctx_step_run_id.reset(step_token)
        ctx_task_retry_count.reset(retry_token)
        root_logger.removeHandler(handler)
        root_logger.setLevel(previous_level)
        loop.set_debug(previous_debug)

        if log_sent:
            log_sender.publish(STOP_LOOP)
            await consume_task
        else:
            consume_task.cancel()
            with suppress(asyncio.CancelledError):
                await consume_task
