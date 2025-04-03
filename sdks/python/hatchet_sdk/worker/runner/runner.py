import asyncio
import contextvars
import ctypes
import functools
import json
import traceback
from concurrent.futures import ThreadPoolExecutor
from enum import Enum
from multiprocessing import Queue
from threading import Thread, current_thread
from typing import Any, Callable, Dict, Literal, cast, overload

from pydantic import BaseModel

from hatchet_sdk.client import Client
from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.clients.dispatcher.action_listener import Action, ActionType
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.durable_event_listener import DurableEventListener
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.context.worker_context import WorkerContext
from hatchet_sdk.contracts.dispatcher_pb2 import (
    GROUP_KEY_EVENT_TYPE_COMPLETED,
    GROUP_KEY_EVENT_TYPE_FAILED,
    GROUP_KEY_EVENT_TYPE_STARTED,
    STEP_EVENT_TYPE_COMPLETED,
    STEP_EVENT_TYPE_FAILED,
    STEP_EVENT_TYPE_STARTED,
)
from hatchet_sdk.exceptions import NonRetryableException
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.contextvars import (
    ctx_step_run_id,
    ctx_worker_id,
    ctx_workflow_run_id,
    spawn_index_lock,
    workflow_spawn_indices,
)
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import R, TWorkflowInput
from hatchet_sdk.utils.typing import WorkflowValidator
from hatchet_sdk.worker.action_listener_process import ActionEvent
from hatchet_sdk.worker.runner.utils.capture_logs import copy_context_vars


class WorkerStatus(Enum):
    INITIALIZED = 1
    STARTING = 2
    HEALTHY = 3
    UNHEALTHY = 4


class Runner:
    def __init__(
        self,
        event_queue: "Queue[ActionEvent]",
        config: ClientConfig,
        slots: int | None = None,
        handle_kill: bool = True,
        action_registry: dict[str, Task[TWorkflowInput, R]] = {},
        validator_registry: dict[str, WorkflowValidator] = {},
        labels: dict[str, str | int] = {},
    ):
        # We store the config so we can dynamically create clients for the dispatcher client.
        self.config = config
        self.client = Client(config)
        self.slots = slots
        self.tasks: dict[str, asyncio.Task[Any]] = {}  # Store run ids and futures
        self.contexts: dict[str, Context] = {}  # Store run ids and contexts
        self.action_registry = action_registry
        self.validator_registry = validator_registry

        self.event_queue = event_queue

        # The thread pool is used for synchronous functions which need to run concurrently
        self.thread_pool = ThreadPoolExecutor(max_workers=slots)
        self.threads: Dict[str, Thread] = {}  # Store run ids and threads

        self.killing = False
        self.handle_kill = handle_kill

        # We need to initialize a new admin and dispatcher client *after* we've started the event loop,
        # otherwise the grpc.aio methods will use a different event loop and we'll get a bunch of errors.
        self.dispatcher_client = DispatcherClient(self.config)
        self.admin_client = AdminClient(self.config)
        self.workflow_run_event_listener = RunEventListenerClient(self.config)
        self.client.workflow_listener = PooledWorkflowRunListener(self.config)
        self.durable_event_listener = DurableEventListener(self.config)

        self.worker_context = WorkerContext(
            labels=labels, client=Client(config=config).dispatcher
        )

    def create_workflow_run_url(self, action: Action) -> str:
        return f"{self.config.server_url}/workflow-runs/{action.workflow_run_id}?tenant={action.tenant_id}"

    def run(self, action: Action) -> None:
        if self.worker_context.id() is None:
            self.worker_context._worker_id = action.worker_id

        match action.action_type:
            case ActionType.START_STEP_RUN:
                log = f"run: start step: {action.action_id}/{action.step_run_id}"
                logger.info(log)
                asyncio.create_task(self.handle_start_step_run(action))
            case ActionType.CANCEL_STEP_RUN:
                log = f"cancel: step run:  {action.action_id}/{action.step_run_id}"
                logger.info(log)
                asyncio.create_task(self.handle_cancel_action(action.step_run_id))
            case ActionType.START_GET_GROUP_KEY:
                log = f"run: get group key:  {action.action_id}/{action.get_group_key_run_id}"
                logger.info(log)
                asyncio.create_task(self.handle_start_group_key_run(action))
            case _:
                log = f"unknown action type: {action.action_type}"
                logger.error(log)

    def step_run_callback(
        self, action: Action, action_task: "Task[TWorkflowInput, R]"
    ) -> Callable[[asyncio.Task[Any]], None]:
        def inner_callback(task: asyncio.Task[Any]) -> None:
            self.cleanup_run_id(action.step_run_id)

            errored = False
            cancelled = task.cancelled()
            output = None

            # Get the output from the future
            try:
                if not cancelled:
                    output = task.result()
            except Exception as e:
                errored = True

                should_not_retry = isinstance(e, NonRetryableException)

                # This except is coming from the application itself, so we want to send that to the Hatchet instance
                self.event_queue.put(
                    ActionEvent(
                        action=action,
                        type=STEP_EVENT_TYPE_FAILED,
                        payload=str(pretty_format_exception(f"{e}", e)),
                        should_not_retry=should_not_retry,
                    )
                )

                logger.error(
                    f"failed step run: {action.action_id}/{action.step_run_id}"
                )

            if not errored and not cancelled:
                self.event_queue.put(
                    ActionEvent(
                        action=action,
                        type=STEP_EVENT_TYPE_COMPLETED,
                        payload=self.serialize_output(output),
                        should_not_retry=False,
                    )
                )

                logger.info(
                    f"finished step run: {action.action_id}/{action.step_run_id}"
                )

        return inner_callback

    def group_key_run_callback(
        self, action: Action
    ) -> Callable[[asyncio.Task[Any]], None]:
        def inner_callback(task: asyncio.Task[Any]) -> None:
            self.cleanup_run_id(action.get_group_key_run_id)

            errored = False
            cancelled = task.cancelled()
            output = None

            # Get the output from the future
            try:
                if not cancelled:
                    output = task.result()
            except Exception as e:
                errored = True
                self.event_queue.put(
                    ActionEvent(
                        action=action,
                        type=GROUP_KEY_EVENT_TYPE_FAILED,
                        payload=str(pretty_format_exception(f"{e}", e)),
                        should_not_retry=False,
                    )
                )

                logger.error(
                    f"failed step run: {action.action_id}/{action.step_run_id}"
                )

            if not errored and not cancelled:
                self.event_queue.put(
                    ActionEvent(
                        action=action,
                        type=GROUP_KEY_EVENT_TYPE_COMPLETED,
                        payload=self.serialize_output(output),
                        should_not_retry=False,
                    )
                )

                logger.info(
                    f"finished step run: {action.action_id}/{action.step_run_id}"
                )

        return inner_callback

    def thread_action_func(
        self, ctx: Context, task: Task[TWorkflowInput, R], action: Action
    ) -> R:
        if action.step_run_id:
            self.threads[action.step_run_id] = current_thread()
        elif action.get_group_key_run_id:
            self.threads[action.get_group_key_run_id] = current_thread()

        return task.call(ctx)

    # We wrap all actions in an async func
    async def async_wrapped_action_func(
        self,
        ctx: Context,
        task: Task[TWorkflowInput, R],
        action: Action,
        run_id: str,
    ) -> R:
        ctx_step_run_id.set(action.step_run_id)
        ctx_workflow_run_id.set(action.workflow_run_id)
        ctx_worker_id.set(action.worker_id)

        try:
            if task.is_async_function:
                return await task.aio_call(ctx)
            else:
                pfunc = functools.partial(
                    # we must copy the context vars to the new thread, as only asyncio natively supports
                    # contextvars
                    copy_context_vars,
                    contextvars.copy_context().items(),
                    self.thread_action_func,
                    ctx,
                    task,
                    action,
                )

                loop = asyncio.get_event_loop()
                return await loop.run_in_executor(self.thread_pool, pfunc)
        except Exception as e:
            logger.error(
                pretty_format_exception(
                    f"exception raised in action ({action.action_id}, retry={action.retry_count}):\n{e}",
                    e,
                )
            )
            raise e
        finally:
            self.cleanup_run_id(run_id)

    def cleanup_run_id(self, run_id: str | None) -> None:
        if run_id in self.tasks:
            del self.tasks[run_id]

        if run_id in self.threads:
            del self.threads[run_id]

        if run_id in self.contexts:
            del self.contexts[run_id]

    @overload
    def create_context(
        self, action: Action, is_durable: Literal[True] = True
    ) -> DurableContext: ...

    @overload
    def create_context(
        self, action: Action, is_durable: Literal[False] = False
    ) -> Context: ...

    def create_context(
        self, action: Action, is_durable: bool = True
    ) -> Context | DurableContext:
        constructor = DurableContext if is_durable else Context

        return constructor(
            action=action,
            dispatcher_client=self.dispatcher_client,
            admin_client=self.admin_client,
            event_client=self.client.event,
            durable_event_listener=self.durable_event_listener,
            worker=self.worker_context,
            validator_registry=self.validator_registry,
            runs_client=self.client.runs,
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    async def handle_start_step_run(self, action: Action) -> None:
        action_name = action.action_id

        # Find the corresponding action function from the registry
        action_func = self.action_registry.get(action_name)

        if action_func:
            context = self.create_context(
                action, True if action_func.is_durable else False
            )

            self.contexts[action.step_run_id] = context
            self.event_queue.put(
                ActionEvent(
                    action=action,
                    type=STEP_EVENT_TYPE_STARTED,
                    payload="",
                    should_not_retry=False,
                )
            )

            loop = asyncio.get_event_loop()
            task = loop.create_task(
                self.async_wrapped_action_func(
                    context, action_func, action, action.step_run_id
                )
            )

            task.add_done_callback(self.step_run_callback(action, action_func))
            self.tasks[action.step_run_id] = task

            try:
                await task
            except Exception:
                # do nothing, this should be caught in the callback
                pass

        ## Once the step run completes, we need to remove the workflow spawn index
        ## so we don't leak memory
        if action.workflow_run_id in workflow_spawn_indices:
            async with spawn_index_lock:
                workflow_spawn_indices.pop(action.workflow_run_id)

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    async def handle_start_group_key_run(self, action: Action) -> Exception | None:
        action_name = action.action_id
        context = self.create_context(action)

        self.contexts[action.get_group_key_run_id] = context

        # Find the corresponding action function from the registry
        action_func = self.action_registry.get(action_name)

        if action_func:
            # send an event that the group key run has started
            self.event_queue.put(
                ActionEvent(
                    action=action,
                    type=GROUP_KEY_EVENT_TYPE_STARTED,
                    payload="",
                    should_not_retry=False,
                )
            )

            loop = asyncio.get_event_loop()
            task = loop.create_task(
                self.async_wrapped_action_func(
                    context, action_func, action, action.get_group_key_run_id
                )
            )

            task.add_done_callback(self.group_key_run_callback(action))
            self.tasks[action.get_group_key_run_id] = task

            try:
                await task
            except Exception as e:
                return e

        return None

    def force_kill_thread(self, thread: Thread) -> None:
        """Terminate a python threading.Thread."""
        try:
            if not thread.is_alive():
                return

            ident = cast(int, thread.ident)

            logger.info(f"Forcefully terminating thread {ident}")

            exc = ctypes.py_object(SystemExit)
            res = ctypes.pythonapi.PyThreadState_SetAsyncExc(ctypes.c_long(ident), exc)
            if res == 0:
                raise ValueError("Invalid thread ID")
            elif res != 1:
                logger.error("PyThreadState_SetAsyncExc failed")

                # Call with exception set to 0 is needed to cleanup properly.
                ctypes.pythonapi.PyThreadState_SetAsyncExc(thread.ident, 0)
                raise SystemError("PyThreadState_SetAsyncExc failed")

            logger.info(f"Successfully terminated thread {ident}")

            # Immediately add a new thread to the thread pool, because we've actually killed a worker
            # in the ThreadPoolExecutor
            self.thread_pool.submit(lambda: None)
        except Exception as e:
            logger.exception(f"Failed to terminate thread: {e}")

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    async def handle_cancel_action(self, run_id: str) -> None:
        try:
            # call cancel to signal the context to stop
            if run_id in self.contexts:
                context = self.contexts.get(run_id)

                if context:
                    await context.aio_cancel()

            await asyncio.sleep(1)

            if run_id in self.tasks:
                future = self.tasks.get(run_id)

                if future:
                    future.cancel()

            # check if thread is still running, if so, print a warning
            if run_id in self.threads:
                thread = self.threads.get(run_id)
                if thread and self.client.config.enable_force_kill_sync_threads:
                    self.force_kill_thread(thread)
                    await asyncio.sleep(1)

                logger.warning(
                    f"Thread {self.threads[run_id].ident} with run id {run_id} is still running after cancellation. This could cause the thread pool to get blocked and prevent new tasks from running."
                )
        finally:
            self.cleanup_run_id(run_id)

    def serialize_output(self, output: Any) -> str:
        if isinstance(output, BaseModel):
            return output.model_dump_json()

        if output is not None:
            try:
                return json.dumps(output, default=str)
            except Exception as e:
                logger.error(f"Could not serialize output: {e}")
                return str(output)

        return ""

    async def wait_for_tasks(self) -> None:
        running = len(self.tasks.keys())
        while running > 0:
            logger.info(f"waiting for {running} tasks to finish...")
            await asyncio.sleep(1)
            running = len(self.tasks.keys())


def pretty_format_exception(message: str, e: Exception) -> str:
    trace = "".join(traceback.format_exception(type(e), e, e.__traceback__))
    return f"{message}\n{trace}"
