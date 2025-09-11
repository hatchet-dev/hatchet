import asyncio
import logging
import signal
import time
from dataclasses import dataclass, field
from multiprocessing import Queue
from typing import Any, List, Mapping, Optional

import grpc

from hatchet_sdk.contracts.dispatcher_pb2 import (
    GROUP_KEY_EVENT_TYPE_STARTED,
    STEP_EVENT_TYPE_STARTED,
    ActionType,
)
from hatchet_sdk.logger import logger
from hatchet_sdk.v0.client import Client, new_client_raw
from hatchet_sdk.v0.clients.dispatcher.action_listener import Action
from hatchet_sdk.v0.clients.dispatcher.dispatcher import (
    ActionListener,
    GetActionListenerRequest,
    new_dispatcher,
)
from hatchet_sdk.v0.clients.rest.models.update_worker_request import UpdateWorkerRequest
from hatchet_sdk.v0.loader import ClientConfig
from hatchet_sdk.v0.utils.backoff import exp_backoff_sleep

ACTION_EVENT_RETRY_COUNT = 5


@dataclass
class ActionEvent:
    action: Action
    type: Any  # TODO type
    payload: Optional[str] = None


STOP_LOOP = "STOP_LOOP"  # Sentinel object to stop the loop

# TODO link to a block post
BLOCKED_THREAD_WARNING = (
    "THE TIME TO START THE STEP RUN IS TOO LONG, THE MAIN THREAD MAY BE BLOCKED"
)


def noop_handler():
    pass


@dataclass
class WorkerActionListenerProcess:
    name: str
    actions: List[str]
    max_runs: int
    config: ClientConfig
    action_queue: Queue
    event_queue: Queue
    handle_kill: bool = True
    debug: bool = False
    labels: dict = field(default_factory=dict)

    listener: ActionListener = field(init=False, default=None)

    killing: bool = field(init=False, default=False)

    action_loop_task: asyncio.Task = field(init=False, default=None)
    event_send_loop_task: asyncio.Task = field(init=False, default=None)

    running_step_runs: Mapping[str, float] = field(init=False, default_factory=dict)

    def __post_init__(self):
        if self.debug:
            logger.setLevel(logging.DEBUG)

        self.client = new_client_raw(self.config, self.debug)

        loop = asyncio.get_event_loop()
        loop.add_signal_handler(
            signal.SIGINT, lambda: asyncio.create_task(self.pause_task_assignment())
        )
        loop.add_signal_handler(
            signal.SIGTERM, lambda: asyncio.create_task(self.pause_task_assignment())
        )
        loop.add_signal_handler(
            signal.SIGQUIT, lambda: asyncio.create_task(self.exit_gracefully())
        )

    async def start(self, retry_attempt=0):
        if retry_attempt > 5:
            logger.error("could not start action listener")
            return

        logger.debug(f"starting action listener: {self.name}")

        try:
            self.dispatcher_client = new_dispatcher(self.config)

            self.listener = await self.dispatcher_client.get_action_listener(
                GetActionListenerRequest(
                    worker_name=self.name,
                    services=["default"],
                    actions=self.actions,
                    max_runs=self.max_runs,
                    _labels=self.labels,
                )
            )

            logger.debug(f"acquired action listener: {self.listener.worker_id}")
        except grpc.RpcError as rpc_error:
            logger.error(f"could not start action listener: {rpc_error}")
            return

        # Start both loops as background tasks
        self.action_loop_task = asyncio.create_task(self.start_action_loop())
        self.event_send_loop_task = asyncio.create_task(self.start_event_send_loop())
        self.blocked_main_loop = asyncio.create_task(self.start_blocked_main_loop())

    # TODO move event methods to separate class
    async def _get_event(self):
        loop = asyncio.get_running_loop()
        return await loop.run_in_executor(None, self.event_queue.get)

    async def start_event_send_loop(self):
        while True:
            event: ActionEvent = await self._get_event()
            if event == STOP_LOOP:
                logger.debug("stopping event send loop...")
                break

            logger.debug(f"tx: event: {event.action.action_id}/{event.type}")
            asyncio.create_task(self.send_event(event))

    async def start_blocked_main_loop(self):
        threshold = 1
        while not self.killing:
            count = 0
            for step_run_id, start_time in self.running_step_runs.items():
                diff = self.now() - start_time
                if diff > threshold:
                    count += 1

            if count > 0:
                logger.warning(f"{BLOCKED_THREAD_WARNING}: Waiting Steps {count}")
            await asyncio.sleep(1)

    async def send_event(self, event: ActionEvent, retry_attempt: int = 1):
        try:
            match event.action.action_type:
                # FIXME: all events sent from an execution of a function are of type ActionType.START_STEP_RUN since
                # the action is re-used. We should change this.
                case ActionType.START_STEP_RUN:
                    # TODO right now we're sending two start_step_run events
                    # one on the action loop and one on the event loop
                    # ideally we change the first to an ack to set the time
                    if event.type == STEP_EVENT_TYPE_STARTED:
                        if event.action.step_run_id in self.running_step_runs:
                            diff = (
                                self.now()
                                - self.running_step_runs[event.action.step_run_id]
                            )
                            if diff > 0.1:
                                logger.warning(
                                    f"{BLOCKED_THREAD_WARNING}: time to start: {diff}s"
                                )
                            else:
                                logger.debug(f"start time: {diff}")
                            del self.running_step_runs[event.action.step_run_id]
                        else:
                            self.running_step_runs[event.action.step_run_id] = (
                                self.now()
                            )

                    asyncio.create_task(
                        self.dispatcher_client.send_step_action_event(
                            event.action, event.type, event.payload
                        )
                    )
                case ActionType.CANCEL_STEP_RUN:
                    logger.debug("unimplemented event send")
                case ActionType.START_GET_GROUP_KEY:
                    asyncio.create_task(
                        self.dispatcher_client.send_group_key_action_event(
                            event.action, event.type, event.payload
                        )
                    )
                case _:
                    logger.error("unknown action type for event send")
        except Exception as e:
            logger.error(
                f"could not send action event ({retry_attempt}/{ACTION_EVENT_RETRY_COUNT}): {e}"
            )
            if retry_attempt <= ACTION_EVENT_RETRY_COUNT:
                await exp_backoff_sleep(retry_attempt, 1)
                await self.send_event(event, retry_attempt + 1)

    def now(self):
        return time.time()

    async def start_action_loop(self):
        try:
            async for action in self.listener:
                if action is None:
                    break

                # Process the action here
                match action.action_type:
                    case ActionType.START_STEP_RUN:
                        self.event_queue.put(
                            ActionEvent(
                                action=action,
                                type=STEP_EVENT_TYPE_STARTED,  # TODO ack type
                            )
                        )
                        logger.info(
                            f"rx: start step run: {action.step_run_id}/{action.action_id}"
                        )

                        # TODO handle this case better...
                        if action.step_run_id in self.running_step_runs:
                            logger.warning(
                                f"step run already running: {action.step_run_id}"
                            )

                    case ActionType.CANCEL_STEP_RUN:
                        logger.info(f"rx: cancel step run: {action.step_run_id}")
                    case ActionType.START_GET_GROUP_KEY:
                        self.event_queue.put(
                            ActionEvent(
                                action=action,
                                type=GROUP_KEY_EVENT_TYPE_STARTED,  # TODO ack type
                            )
                        )
                        logger.info(
                            f"rx: start group key: {action.get_group_key_run_id}"
                        )
                    case _:
                        logger.error(
                            f"rx: unknown action type ({action.action_type}): {action.action_type}"
                        )
                try:
                    self.action_queue.put(action)
                except Exception as e:
                    logger.error(f"error putting action: {e}")

        except Exception as e:
            logger.error(f"error in action loop: {e}")
        finally:
            logger.info("action loop closed")
            if not self.killing:
                await self.exit_gracefully(skip_unregister=True)

    async def cleanup(self):
        self.killing = True

        if self.listener is not None:
            self.listener.cleanup()

        self.event_queue.put(STOP_LOOP)

    async def pause_task_assignment(self) -> None:
        await self.client.rest.aio.worker_api.worker_update(
            worker=self.listener.worker_id,
            update_worker_request=UpdateWorkerRequest(isPaused=True),
        )

    async def exit_gracefully(self, skip_unregister=False):
        await self.pause_task_assignment()

        if self.killing:
            return

        logger.debug("closing action listener...")

        await self.cleanup()

        while not self.event_queue.empty():
            pass

        logger.info("action listener closed")

    def exit_forcefully(self):
        asyncio.run(self.cleanup())
        logger.debug("forcefully closing listener...")


def worker_action_listener_process(*args, **kwargs):
    async def run():
        process = WorkerActionListenerProcess(*args, **kwargs)
        await process.start()
        # Keep the process running
        while not process.killing:
            await asyncio.sleep(0.1)

    asyncio.run(run())
