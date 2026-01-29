import asyncio
import ctypes
import functools
import json
from collections.abc import Callable
from concurrent.futures import ThreadPoolExecutor
from dataclasses import asdict, is_dataclass
from enum import Enum
from multiprocessing import Queue
from textwrap import dedent
from threading import Thread, current_thread
from typing import Any, Literal, cast, overload

from pydantic import BaseModel

from hatchet_sdk.client import Client
from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.listeners.durable_event_listener import DurableEventListener
from hatchet_sdk.clients.listeners.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.context.worker_context import WorkerContext
from hatchet_sdk.contracts.dispatcher_pb2 import (
    STEP_EVENT_TYPE_COMPLETED,
    STEP_EVENT_TYPE_FAILED,
    STEP_EVENT_TYPE_STARTED,
)
from hatchet_sdk.exceptions import (
    IllegalTaskOutputError,
    NonRetryableException,
    TaskRunError,
)
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import Action, ActionKey, ActionType
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_additional_metadata,
    ctx_step_run_id,
    ctx_task_retry_count,
    ctx_worker_id,
    ctx_workflow_run_id,
    spawn_index_lock,
    task_count,
    workflow_spawn_indices,
)
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import R, TWorkflowInput
from hatchet_sdk.serde import HATCHET_PYDANTIC_SENTINEL
from hatchet_sdk.utils.cache import BoundedDict
from hatchet_sdk.utils.serde import remove_null_unicode_character
from hatchet_sdk.utils.typing import DataclassInstance
from hatchet_sdk.worker.action_listener_process import ActionEvent
from hatchet_sdk.worker.runner.utils.capture_logs import (
    AsyncLogSender,
    ContextVarToCopy,
    ContextVarToCopyDict,
    ContextVarToCopyInt,
    ContextVarToCopyStr,
    copy_context_vars,
)


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
        slots: int,
        handle_kill: bool,
        action_registry: dict[str, Task[TWorkflowInput, R]],
        labels: dict[str, str | int] | None,
        lifespan_context: Any | None,
        log_sender: AsyncLogSender,
    ):
        # We store the config so we can dynamically create clients for the dispatcher client.
        self.config = config

        self.slots = slots
        self.tasks: dict[ActionKey, asyncio.Task[Any]] = {}  # Store run ids and futures
        self.contexts: dict[ActionKey, Context] = {}  # Store run ids and contexts
        self.cancellations = BoundedDict[str, bool](maxsize=1000)
        self.action_registry = action_registry or {}

        self.event_queue = event_queue

        # The thread pool is used for synchronous functions which need to run concurrently
        self.thread_pool = ThreadPoolExecutor(max_workers=slots)
        self.threads: dict[ActionKey, Thread] = {}  # Store run ids and threads
        self.running_tasks = set[asyncio.Task[Exception | None]]()

        self.killing = False
        self.handle_kill = handle_kill

        self.dispatcher_client = DispatcherClient(self.config)
        self.workflow_run_event_listener = RunEventListenerClient(self.config)
        self.workflow_listener = PooledWorkflowRunListener(self.config)
        self.admin_client = AdminClient(
            self.config,
            self.workflow_listener,
            self.workflow_run_event_listener,
        )

        self.runs_client = RunsClient(
            config=self.config,
            workflow_run_event_listener=self.workflow_run_event_listener,
            workflow_run_listener=self.workflow_listener,
            admin_client=self.admin_client,
        )
        self.event_client = EventClient(self.config)
        self.durable_event_listener = DurableEventListener(self.config)

        self.worker_context = WorkerContext(
            labels=labels or {}, client=Client(config=config).dispatcher
        )

        self.lifespan_context = lifespan_context
        self.log_sender = log_sender

        if self.config.enable_thread_pool_monitoring:
            self.start_background_monitoring()

    def create_workflow_run_url(self, action: Action) -> str:
        return f"{self.config.server_url}/workflow-runs/{action.workflow_run_id}?tenant={action.tenant_id}"

    def run(self, action: Action) -> None:
        if self.worker_context.id() is None:
            self.worker_context._worker_id = action.worker_id

        t: asyncio.Task[Exception | None] | None = None
        match action.action_type:
            case ActionType.START_STEP_RUN:
                log = f"run: start step: {action.action_id}/{action.step_run_id}"
                logger.info(log)
                t = asyncio.create_task(self.handle_start_step_run(action))
            case ActionType.CANCEL_STEP_RUN:
                log = f"cancel: step run:  {action.action_id}/{action.step_run_id}/{action.retry_count}"
                logger.info(log)
                t = asyncio.create_task(self.handle_cancel_action(action))
            case _:
                log = f"unknown action type: {action.action_type}"
                logger.error(log)

        if t is not None:
            self.running_tasks.add(t)
            t.add_done_callback(lambda task: self.running_tasks.discard(task))

    def step_run_callback(
        self, action: Action, t: Task[TWorkflowInput, R]
    ) -> Callable[[asyncio.Task[Any]], None]:
        def inner_callback(task: asyncio.Task[Any]) -> None:
            self.cleanup_run_id(action.key)
            was_cancelled = self.cancellations.pop(action.key, False)

            if was_cancelled or task.cancelled():
                return

            try:
                output = task.result()
            except Exception as e:
                should_not_retry = isinstance(e, NonRetryableException)

                exc = TaskRunError.from_exception(e, action.step_run_id)

                # This except is coming from the application itself, so we want to send that to the Hatchet instance
                self.event_queue.put(
                    ActionEvent(
                        action=action,
                        type=STEP_EVENT_TYPE_FAILED,
                        payload=exc.serialize(include_metadata=True),
                        should_not_retry=should_not_retry,
                    )
                )

                # log as info if we're going to retry or we explicitly should _not_ retry
                # so that e.g. Sentry does not get reported multiple exceptions from multiple retries of a single task
                log_as_info = should_not_retry or action.retry_count < t.retries

                log_with_level = logger.info if log_as_info else logger.exception

                log_with_level(
                    f"failed step run: {action.action_id}/{action.step_run_id}\n{exc.serialize(include_metadata=False)}"
                )

                return

            try:
                output = self.serialize_output(output)

                self.event_queue.put(
                    ActionEvent(
                        action=action,
                        type=STEP_EVENT_TYPE_COMPLETED,
                        payload=output,
                        should_not_retry=False,
                    )
                )
            except IllegalTaskOutputError as e:
                exc = TaskRunError.from_exception(e, action.step_run_id)
                self.event_queue.put(
                    ActionEvent(
                        action=action,
                        type=STEP_EVENT_TYPE_FAILED,
                        payload=exc.serialize(include_metadata=True),
                        should_not_retry=False,
                    )
                )

                logger.exception(
                    f"failed step run: {action.action_id}/{action.step_run_id}\n{exc.serialize(include_metadata=False)}"
                )

                return

            logger.info(f"finished step run: {action.action_id}/{action.step_run_id}")

        return inner_callback

    def thread_action_func(
        self,
        ctx: Context,
        task: Task[TWorkflowInput, R],
        action: Action,
        dependencies: dict[str, Any],
    ) -> R:
        if action.step_run_id:
            self.threads[action.key] = current_thread()

        return task.call(ctx, dependencies)

    # We wrap all actions in an async func
    async def async_wrapped_action_func(
        self,
        ctx: Context,
        task: Task[TWorkflowInput, R],
        action: Action,
    ) -> R:
        ctx_step_run_id.set(action.step_run_id)
        ctx_workflow_run_id.set(action.workflow_run_id)
        ctx_worker_id.set(action.worker_id)
        ctx_action_key.set(action.key)
        ctx_additional_metadata.set(action.additional_metadata)
        ctx_task_retry_count.set(action.retry_count)

        async with task._unpack_dependencies_with_cleanup(ctx) as dependencies:
            try:
                if task.is_async_function:
                    return await task.aio_call(ctx, dependencies)

                pfunc = functools.partial(
                    # we must copy the context vars to the new thread, as only asyncio natively supports
                    # contextvars
                    copy_context_vars,
                    [
                        ContextVarToCopy(
                            var=ContextVarToCopyStr(
                                name="ctx_step_run_id",
                                value=action.step_run_id,
                            )
                        ),
                        ContextVarToCopy(
                            var=ContextVarToCopyStr(
                                name="ctx_workflow_run_id",
                                value=action.workflow_run_id,
                            )
                        ),
                        ContextVarToCopy(
                            var=ContextVarToCopyStr(
                                name="ctx_worker_id",
                                value=action.worker_id,
                            )
                        ),
                        ContextVarToCopy(
                            var=ContextVarToCopyStr(
                                name="ctx_action_key",
                                value=action.key,
                            )
                        ),
                        ContextVarToCopy(
                            var=ContextVarToCopyDict(
                                name="ctx_additional_metadata",
                                value=action.additional_metadata,
                            )
                        ),
                        ContextVarToCopy(
                            var=ContextVarToCopyInt(
                                name="ctx_task_retry_count",
                                value=action.retry_count,
                            )
                        ),
                    ],
                    self.thread_action_func,
                    ctx,
                    task,
                    action,
                    dependencies,
                )

                loop = asyncio.get_event_loop()
                return await loop.run_in_executor(self.thread_pool, pfunc)
            finally:
                self.cleanup_run_id(action.key)

    async def log_thread_pool_status(self) -> None:
        thread_pool_details = {
            "max_workers": self.slots,
            "total_threads": len(self.thread_pool._threads),
            "idle_threads": self.thread_pool._idle_semaphore._value,
            "active_threads": len(self.threads),
            "pending_tasks": len(self.tasks),
            "queue_size": self.thread_pool._work_queue.qsize(),
            "threads_alive": sum(1 for t in self.thread_pool._threads if t.is_alive()),
            "threads_daemon": sum(1 for t in self.thread_pool._threads if t.daemon),
        }

        logger.warning("thread pool detailed status %s", thread_pool_details)

    async def _start_monitoring(self) -> None:
        logger.debug("thread pool monitoring started")
        try:
            while True:
                await self.log_thread_pool_status()

                for key in self.threads:
                    if key not in self.tasks:
                        logger.debug(f"potential zombie thread found for key {key}")

                for key, task in self.tasks.items():
                    if task.done() and key in self.threads:
                        logger.debug(
                            f"task is done but thread still exists for key {key}"
                        )

                await asyncio.sleep(60)
        except asyncio.CancelledError:
            logger.warning("thread pool monitoring task cancelled")
        except Exception as e:
            logger.exception(f"error in thread pool monitoring: {e}")

    def start_background_monitoring(self) -> None:
        loop = asyncio.get_event_loop()
        self.monitoring_task = loop.create_task(self._start_monitoring())
        logger.debug("started thread pool monitoring background task")

    def cleanup_run_id(self, key: ActionKey) -> None:
        if key in self.tasks:
            del self.tasks[key]

        if key in self.threads:
            del self.threads[key]

        if key in self.contexts:
            if self.contexts[key].exit_flag:
                self.cancellations[key] = True

            del self.contexts[key]

    @overload
    def create_context(
        self, action: Action, max_attempts: int, is_durable: Literal[True] = True
    ) -> DurableContext: ...

    @overload
    def create_context(
        self, action: Action, max_attempts: int, is_durable: Literal[False] = False
    ) -> Context: ...

    def create_context(
        self,
        action: Action,
        max_attempts: int,
        is_durable: bool = True,
    ) -> Context | DurableContext:
        constructor = DurableContext if is_durable else Context

        return constructor(
            action=action,
            dispatcher_client=self.dispatcher_client,
            admin_client=self.admin_client,
            event_client=self.event_client,
            durable_event_listener=self.durable_event_listener,
            worker=self.worker_context,
            runs_client=self.runs_client,
            lifespan_context=self.lifespan_context,
            log_sender=self.log_sender,
            max_attempts=max_attempts,
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    async def handle_start_step_run(self, action: Action) -> Exception | None:
        action_name = action.action_id

        # Find the corresponding action function from the registry
        action_func = self.action_registry.get(action_name)

        if action_func:
            context = self.create_context(
                action=action,
                max_attempts=action_func.retries + 1,
                is_durable=True if action_func.is_durable else False,  # noqa: SIM210
            )

            self.contexts[action.key] = context
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
                self.async_wrapped_action_func(context, action_func, action)
            )

            task.add_done_callback(self.step_run_callback(action, action_func))
            self.tasks[action.key] = task

            task_count.increment()

            ## FIXME: Handle cancelled exceptions and other special exceptions
            ## that we don't want to suppress here
            try:
                await task
            except Exception as e:
                ## Used for the OTel instrumentor to capture exceptions
                return e

        ## Once the step run completes, we need to remove the workflow spawn index
        ## so we don't leak memory
        if action.key in workflow_spawn_indices:
            async with spawn_index_lock:
                workflow_spawn_indices.pop(action.key)

        return None

    def force_kill_thread(self, thread: Thread) -> None:
        """Terminate a python threading.Thread."""
        try:
            if not thread.is_alive():
                return

            ident = cast(int, thread.ident)

            logger.info(f"forcefully terminating thread {ident}")

            exc = ctypes.py_object(SystemExit)
            res = ctypes.pythonapi.PyThreadState_SetAsyncExc(ctypes.c_long(ident), exc)
            if res == 0:
                raise ValueError("Invalid thread ID")
            if res != 1:
                logger.error("PyThreadState_SetAsyncExc failed")

                # Call with exception set to 0 is needed to cleanup properly.
                ctypes.pythonapi.PyThreadState_SetAsyncExc(thread.ident, 0)
                raise SystemError("PyThreadState_SetAsyncExc failed")

            logger.info(f"successfully terminated thread {ident}")

            # Immediately add a new thread to the thread pool, because we've actually killed a worker
            # in the ThreadPoolExecutor
            self.thread_pool.submit(lambda: None)
        except Exception as e:
            logger.exception(f"failed to terminate thread: {e}")

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    async def handle_cancel_action(self, action: Action) -> None:
        key = action.key
        try:
            # call cancel to signal the context to stop
            if key in self.contexts:
                self.contexts[key]._set_cancellation_flag()
                self.cancellations[key] = True

            await asyncio.sleep(1)

            if key in self.tasks:
                self.tasks[key].cancel()

            # check if thread is still running, if so, print a warning
            if key in self.threads:
                thread = self.threads[key]

                if self.config.enable_force_kill_sync_threads:
                    self.force_kill_thread(thread)
                    await asyncio.sleep(1)

                logger.warning(
                    f"thread {self.threads[key].ident} with key {key} is still running after cancellation. This could cause the thread pool to get blocked and prevent new tasks from running."
                )
        finally:
            self.cleanup_run_id(key)

    def serialize_output(self, output: Any) -> str:
        if not output:
            return ""

        if isinstance(output, BaseModel):
            try:
                output = output.model_dump(
                    mode="json", context=HATCHET_PYDANTIC_SENTINEL
                )
            except Exception as e:
                logger.exception("could not serialize pydantic model output")

                raise IllegalTaskOutputError(
                    f"could not serialize Pydantic BaseModel output: {e}"
                ) from e
        elif is_dataclass(output):
            output = asdict(cast(DataclassInstance, output))

        if not isinstance(output, dict):
            raise IllegalTaskOutputError(
                f"Tasks must return either a dictionary, a Pydantic BaseModel, or a dataclass which can be serialized to a JSON object. Got object of type {type(output)} instead."
            )

        if output is None:
            return ""

        try:
            serialized_output = json.dumps(output, default=str)
        except Exception as e:
            logger.exception("could not serialize output")
            raise IllegalTaskOutputError(
                "Task output could not be serialized to JSON. Please ensure that all task outputs are JSON serializable."
            ) from e

        if "\\u0000" in serialized_output:
            raise IllegalTaskOutputError(
                dedent(
                    f"""
                Task outputs cannot contain the unicode null character \\u0000

                Please see this Discord thread: https://discord.com/channels/1088927970518909068/1384324576166678710/1386714014565928992
                Relevant Postgres documentation: https://www.postgresql.org/docs/current/datatype-json.html

                Use `hatchet_sdk.{remove_null_unicode_character.__name__}` to sanitize your output if you'd like to remove the character.
                """
                )
            )

        return serialized_output

    async def wait_for_tasks(self) -> None:
        running = len(self.tasks.keys())
        while running > 0:
            logger.info(f"waiting for {running} tasks to finish...")
            await asyncio.sleep(1)
            running = len(self.tasks.keys())
