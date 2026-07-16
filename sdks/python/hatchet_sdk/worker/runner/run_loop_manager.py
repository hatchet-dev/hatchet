import asyncio
import logging
from typing import TYPE_CHECKING, Any, TypeVar

from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import Action
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.types.labels import WorkerLabel
from hatchet_sdk.utils.typing import STOP_LOOP, STOP_LOOP_TYPE
from hatchet_sdk.worker.runner.runner import Runner
from hatchet_sdk.worker.runner.utils.capture_logs import AsyncLogSender, capture_logs

if TYPE_CHECKING:
    from multiprocessing import Queue

    from hatchet_sdk.worker.action_listener_process import ActionEvent

T = TypeVar("T")


class WorkerActionRunLoopManager:
    def __init__(
        self,
        name: str,
        action_registry: dict[str, Task[Any, Any]],
        slots: int,
        durable_slots: int,
        config: ClientConfig,
        action_queue: "Queue[Action | STOP_LOOP_TYPE]",
        event_queue: "Queue[ActionEvent | STOP_LOOP_TYPE]",
        loop: asyncio.AbstractEventLoop,
        handle_kill: bool,
        labels: list[WorkerLabel],
        lifespan_context: Any | None,  # noqa: ANN401
        engine_version: str | None = None,
    ) -> None:
        self.name = name
        self.action_registry = action_registry
        self.slots = slots
        self.durable_slots = durable_slots
        self.config = config
        self.action_queue = action_queue
        self.event_queue = event_queue
        self.loop = loop
        self.handle_kill = handle_kill
        self.labels = labels
        self.lifespan_context = lifespan_context
        self.engine_version = engine_version
        self.config = config

        if self.config.debug:
            logger.setLevel(logging.DEBUG)

        self.killing = False
        self.runner: Runner | None = None

        self._event_client = EventClient(config)
        self.start_loop_manager_task: asyncio.Task[None] | None = None
        self.log_sender = AsyncLogSender(self._event_client)

        self.log_sender.start()
        self.start()

    def start(self) -> None:
        self.start_loop_manager_task = self.loop.create_task(self.aio_start())

    async def aio_start(self) -> None:
        if self.config.disable_log_capture:
            await self._async_start()
        else:
            await capture_logs(
                self.config.logger,
                self.log_sender,
                self._async_start,
            )()

    async def _async_start(self) -> None:
        logger.info("starting action runner...")
        self.loop = asyncio.get_running_loop()
        # needed for graceful termination
        k = self.loop.create_task(self._start_action_loop())
        await k

    def cleanup(self) -> None:
        self.killing = True

        self.action_queue.put(STOP_LOOP)
        self.log_sender.stop()

    async def evict_all_waiting_durable_runs(self) -> None:
        if self.runner:
            await self.runner.evict_all_waiting_durable_runs()

    async def wait_for_tasks(self) -> None:
        if self.runner:
            await self.runner.wait_for_tasks()

    async def _start_action_loop(self) -> None:
        self.runner = Runner(
            self.event_queue,
            self.config,
            self.slots,
            self.durable_slots,
            self.handle_kill,
            self.action_registry,
            self.labels,
            self.lifespan_context,
            self.log_sender,
            engine_version=self.engine_version,
        )

        logger.debug(
            f"'{self.name}' found the following actions registered: {list(self.action_registry.keys())}"
        )
        logger.info(f"'{self.name}' started, waiting for tasks...")
        while not self.killing:
            action = await self._get_action()
            if action == STOP_LOOP:
                logger.debug("stopping action runner loop...")
                break

            self.runner.run(action)
        logger.debug("action runner loop stopped")

    async def _get_action(self) -> Action | STOP_LOOP_TYPE:
        return await self.loop.run_in_executor(None, self.action_queue.get)

    async def exit_gracefully(self) -> None:
        if self.killing:
            return

        logger.info("gracefully exiting runner...")
        await self.evict_all_waiting_durable_runs()
        await self.wait_for_tasks()
        self.cleanup()

        # Wait for 1 second to allow last calls to flush. These are calls which have been
        # added to the event loop as callbacks to tasks, so we're not aware of them in the
        # task list.
        await asyncio.sleep(1)

    def exit_forcefully(self) -> None:
        logger.info("forcefully exiting runner...")
        self.cleanup()
