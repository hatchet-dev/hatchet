import json
import traceback
from concurrent.futures import Future, ThreadPoolExecutor
from datetime import timedelta
from typing import TYPE_CHECKING, Any, cast
from warnings import warn

from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.clients.dispatcher.dispatcher import (  # type: ignore[attr-defined]
    Action,
    DispatcherClient,
)
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.listeners.durable_event_listener import (
    DurableEventListener,
    RegisterDurableEventRequest,
)
from hatchet_sdk.context.worker_context import WorkerContext
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.logger import logger
from hatchet_sdk.utils.timedelta_to_expression import Duration, timedelta_to_expr
from hatchet_sdk.utils.typing import JSONSerializableMapping
from hatchet_sdk.waits import SleepCondition, UserEventCondition

if TYPE_CHECKING:
    from hatchet_sdk.runnables.task import Task
    from hatchet_sdk.runnables.types import R, TWorkflowInput


class Context:
    def __init__(
        self,
        action: Action,
        dispatcher_client: DispatcherClient,
        admin_client: AdminClient,
        event_client: EventClient,
        durable_event_listener: DurableEventListener | None,
        worker: WorkerContext,
        runs_client: RunsClient,
        lifespan_context: Any | None,
    ):
        self.worker = worker

        self.data = action.action_payload

        self.action = action

        self.step_run_id = action.step_run_id
        self.exit_flag = False
        self.dispatcher_client = dispatcher_client
        self.admin_client = admin_client
        self.event_client = event_client
        self.runs_client = runs_client
        self.durable_event_listener = durable_event_listener

        # FIXME: this limits the number of concurrent log requests to 1, which means we can do about
        # 100 log lines per second but this depends on network.
        self.logger_thread_pool = ThreadPoolExecutor(max_workers=1)
        self.stream_event_thread_pool = ThreadPoolExecutor(max_workers=1)

        self.input = self.data.input
        self.filter_payload = self.data.filter_payload

        self._lifespan_context = lifespan_context

    def was_skipped(self, task: "Task[TWorkflowInput, R]") -> bool:
        return self.data.parents.get(task.name, {}).get("skipped", False)

    @property
    def trigger_data(self) -> JSONSerializableMapping:
        return self.data.triggers

    def task_output(self, task: "Task[TWorkflowInput, R]") -> "R":
        from hatchet_sdk.runnables.types import R

        if self.was_skipped(task):
            raise ValueError(f"{task.name} was skipped")

        try:
            parent_step_data = cast(R, self.data.parents[task.name])
        except KeyError:
            raise ValueError(f"Step output for '{task.name}' not found")

        if parent_step_data and (v := task.validators.step_output):
            return cast(R, v.model_validate(parent_step_data))

        return parent_step_data

    def aio_task_output(self, task: "Task[TWorkflowInput, R]") -> "R":
        warn(
            "`aio_task_output` is deprecated. Use `task_output` instead.",
            DeprecationWarning,
        )

        if task.is_async_function:
            return self.task_output(task)

        raise ValueError(
            f"Task '{task.name}' is not an async function. Use `task_output` instead."
        )

    @property
    def was_triggered_by_event(self) -> bool:
        return self.data.triggered_by == "event"

    @property
    def workflow_input(self) -> JSONSerializableMapping:
        return self.input

    @property
    def lifespan(self) -> Any:
        return self._lifespan_context

    @property
    def workflow_run_id(self) -> str:
        return self.action.workflow_run_id

    def _set_cancellation_flag(self) -> None:
        self.exit_flag = True

    def cancel(self) -> None:
        logger.debug("cancelling step...")
        self.runs_client.cancel(self.step_run_id)
        self._set_cancellation_flag()

    async def aio_cancel(self) -> None:
        logger.debug("cancelling step...")
        await self.runs_client.aio_cancel(self.step_run_id)
        self._set_cancellation_flag()

    # done returns true if the context has been cancelled
    def done(self) -> bool:
        return self.exit_flag

    def _log(self, line: str) -> tuple[bool, Exception | None]:
        try:
            self.event_client.log(message=line, step_run_id=self.step_run_id)
            return True, None
        except Exception as e:
            # we don't want to raise an exception here, as it will kill the log thread
            return False, e

    def log(
        self, line: str | JSONSerializableMapping, raise_on_error: bool = False
    ) -> None:
        if self.step_run_id == "":
            return

        if not isinstance(line, str):
            try:
                line = json.dumps(line)
            except Exception:
                line = str(line)

        future = self.logger_thread_pool.submit(self._log, line)

        def handle_result(future: Future[tuple[bool, Exception | None]]) -> None:
            success, exception = future.result()

            if not success and exception:
                if raise_on_error:
                    raise exception
                else:
                    thread_trace = "".join(
                        traceback.format_exception(
                            type(exception), exception, exception.__traceback__
                        )
                    )
                    call_site_trace = "".join(traceback.format_stack())
                    logger.error(
                        f"Error in log thread: {exception}\n{thread_trace}\nCalled from:\n{call_site_trace}"
                    )

        future.add_done_callback(handle_result)

    def release_slot(self) -> None:
        return self.dispatcher_client.release_slot(self.step_run_id)

    def _put_stream(self, data: str | bytes) -> None:
        try:
            self.event_client.stream(data=data, step_run_id=self.step_run_id)
        except Exception as e:
            logger.error(f"Error putting stream event: {e}")

    def put_stream(self, data: str | bytes) -> None:
        if self.step_run_id == "":
            return

        self.stream_event_thread_pool.submit(self._put_stream, data)

    def refresh_timeout(self, increment_by: str | timedelta) -> None:
        if isinstance(increment_by, timedelta):
            increment_by = timedelta_to_expr(increment_by)

        try:
            return self.dispatcher_client.refresh_timeout(
                step_run_id=self.step_run_id, increment_by=increment_by
            )
        except Exception as e:
            logger.error(f"Error refreshing timeout: {e}")

    @property
    def retry_count(self) -> int:
        return self.action.retry_count

    @property
    def additional_metadata(self) -> JSONSerializableMapping | None:
        return self.action.additional_metadata

    @property
    def child_index(self) -> int | None:
        return self.action.child_workflow_index

    @property
    def child_key(self) -> str | None:
        return self.action.child_workflow_key

    @property
    def parent_workflow_run_id(self) -> str | None:
        return self.action.parent_workflow_run_id

    @property
    def priority(self) -> int | None:
        return self.action.priority

    @property
    def workflow_id(self) -> str | None:
        return self.action.workflow_id

    @property
    def workflow_version_id(self) -> str | None:
        return self.action.workflow_version_id

    @property
    def task_run_errors(self) -> dict[str, str]:
        errors = self.data.step_run_errors

        if not errors:
            logger.error(
                "No step run errors found. `context.step_run_errors` is intended to be run in an on-failure step, and will only work on engine versions more recent than v0.53.10"
            )

        return errors

    def fetch_task_run_error(
        self,
        task: "Task[TWorkflowInput, R]",
    ) -> str | None:
        errors = self.data.step_run_errors

        return errors.get(task.name)


class DurableContext(Context):
    async def aio_wait_for(
        self, signal_key: str, *conditions: SleepCondition | UserEventCondition
    ) -> dict[str, Any]:
        if self.durable_event_listener is None:
            raise ValueError("Durable event listener is not available")

        task_id = self.step_run_id

        request = RegisterDurableEventRequest(
            task_id=task_id,
            signal_key=signal_key,
            conditions=list(conditions),
        )

        self.durable_event_listener.register_durable_event(request)

        return await self.durable_event_listener.result(
            task_id,
            signal_key,
        )

    async def aio_sleep_for(self, duration: Duration) -> dict[str, Any]:
        """
        Lightweight wrapper for durable sleep. Allows for shorthand usage of `ctx.aio_wait_for` when specifying a sleep condition.

        For more complicated conditions, use `ctx.aio_wait_for` directly.
        """

        return await self.aio_wait_for(
            f"sleep:{timedelta_to_expr(duration)}", SleepCondition(duration=duration)
        )
