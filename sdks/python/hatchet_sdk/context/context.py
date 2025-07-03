import asyncio
import json
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
from hatchet_sdk.conditions import (
    OrGroup,
    SleepCondition,
    UserEventCondition,
    flatten_conditions,
)
from hatchet_sdk.context.worker_context import WorkerContext
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.logger import logger
from hatchet_sdk.utils.timedelta_to_expression import Duration, timedelta_to_expr
from hatchet_sdk.utils.typing import JSONSerializableMapping
from hatchet_sdk.worker.runner.utils.capture_logs import AsyncLogSender, LogRecord

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
        log_sender: AsyncLogSender,
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

        self.input = self.data.input
        self.filter_payload = self.data.filter_payload
        self.log_sender = log_sender

        self._lifespan_context = lifespan_context

        self.stream_index = 0

    def _increment_stream_index(self) -> int:
        index = self.stream_index
        self.stream_index += 1

        return index

    def was_skipped(self, task: "Task[TWorkflowInput, R]") -> bool:
        """
        Check if a given task was skipped. You can read about skipping in [the docs](https://docs.hatchet.run/home/conditional-workflows#skip_if).

        :param task: The task to check the status of (skipped or not).
        :return: True if the task was skipped, False otherwise.
        """
        return self.data.parents.get(task.name, {}).get("skipped", False) is True

    @property
    def trigger_data(self) -> JSONSerializableMapping:
        return self.data.triggers

    def task_output(self, task: "Task[TWorkflowInput, R]") -> "R":
        """
        Get the output of a parent task in a DAG.

        :param task: The task whose output you want to retrieve.
        :return: The output of the parent task, validated against the task's validators.
        :raises ValueError: If the task was skipped or if the step output for the task is not found.
        """
        from hatchet_sdk.runnables.types import R

        if self.was_skipped(task):
            raise ValueError(f"{task.name} was skipped")

        try:
            parent_step_data = cast(R, self.data.parents[task.name])
        except KeyError as e:
            raise ValueError(f"Step output for '{task.name}' not found") from e

        if parent_step_data and (v := task.validators.step_output):
            return cast(R, v.model_validate(parent_step_data))

        return parent_step_data

    def aio_task_output(self, task: "Task[TWorkflowInput, R]") -> "R":
        warn(
            "`aio_task_output` is deprecated. Use `task_output` instead.",
            DeprecationWarning,
            stacklevel=2,
        )

        if task.is_async_function:
            return self.task_output(task)

        raise ValueError(
            f"Task '{task.name}' is not an async function. Use `task_output` instead."
        )

    @property
    def was_triggered_by_event(self) -> bool:
        """
        A property that indicates whether the workflow was triggered by an event.

        :return: True if the workflow was triggered by an event, False otherwise.
        """
        return self.data.triggered_by == "event"

    @property
    def workflow_input(self) -> JSONSerializableMapping:
        """
        The input to the workflow, as a dictionary. It's recommended to use the `input` parameter to the task (the first argument passed into the task at runtime) instead of this property.

        :return: The input to the workflow.
        """
        return self.input

    @property
    def lifespan(self) -> Any:
        """
        The worker lifespan, if it exists. You can read about lifespans in [the docs](https://docs.hatchet.run/home/lifespans).

        **Note: You'll need to cast the return type of this property to the type returned by your lifespan generator.**
        """
        return self._lifespan_context

    @property
    def workflow_run_id(self) -> str:
        """
        The id of the current workflow run.

        :return: The id of the current workflow run.
        """
        return self.action.workflow_run_id

    def _set_cancellation_flag(self) -> None:
        self.exit_flag = True

    def cancel(self) -> None:
        """
        Cancel the current task run. This will call the Hatchet API to cancel the step run and set the exit flag to True.

        :return: None
        """
        logger.debug("cancelling step...")
        self.runs_client.cancel(self.step_run_id)
        self._set_cancellation_flag()

    async def aio_cancel(self) -> None:
        """
        Cancel the current task run. This will call the Hatchet API to cancel the step run and set the exit flag to True.

        :return: None
        """
        logger.debug("cancelling step...")
        await self.runs_client.aio_cancel(self.step_run_id)
        self._set_cancellation_flag()

    def done(self) -> bool:
        """
        Check if the current task run has been cancelled.

        :return: True if the task run has been cancelled, False otherwise.
        """
        return self.exit_flag

    def log(
        self, line: str | JSONSerializableMapping, raise_on_error: bool = False
    ) -> None:
        """
        Log a line to the Hatchet API. This will send the log line to the Hatchet API and return immediately.

        :param line: The line to log. Can be a string or a JSON serializable mapping.
        :param raise_on_error: If True, will raise an exception if the log fails. Defaults to False.
        :return: None
        """

        if self.step_run_id == "":
            return

        if not isinstance(line, str):
            try:
                line = json.dumps(line)
            except Exception:
                line = str(line)

        logger.info(line)
        self.log_sender.publish(LogRecord(message=line, step_run_id=self.step_run_id))

    def release_slot(self) -> None:
        """
        Manually release the slot for the current step run to free up a slot on the worker. Note that this is an advanced feature and should be used with caution.

        :return: None
        """
        return self.dispatcher_client.release_slot(self.step_run_id)

    def put_stream(self, data: str | bytes) -> None:
        """
        Put a stream event to the Hatchet API. This will send the data to the Hatchet API and return immediately. You can then subscribe to the stream from a separate consumer.

        :param data: The data to send to the Hatchet API. Can be a string or bytes.
        :return: None
        """
        try:
            ix = self._increment_stream_index()

            self.event_client.stream(
                data=data,
                step_run_id=self.step_run_id,
                index=ix,
            )
        except Exception as e:
            logger.error(f"Error putting stream event: {e}")

    async def aio_put_stream(self, data: str | bytes) -> None:
        """
        Put a stream event to the Hatchet API. This will send the data to the Hatchet API and return immediately. You can then subscribe to the stream from a separate consumer.

        :param data: The data to send to the Hatchet API. Can be a string or bytes.
        :return: None
        """
        await asyncio.to_thread(self.put_stream, data)

    def refresh_timeout(self, increment_by: str | timedelta) -> None:
        """
        Refresh the timeout for the current task run. You can read about refreshing timeouts in [the docs](https://docs.hatchet.run/home/timeouts#refreshing-timeouts).

        :param increment_by: The amount of time to increment the timeout by. Can be a string (e.g. "5m") or a timedelta object.
        :return: None
        """
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
        """
        The retry count of the current task run, which corresponds to the number of times the task has been retried.

        :return: The retry count of the current task run.
        """
        return self.action.retry_count

    @property
    def attempt_number(self) -> int:
        """
        The attempt number of the current task run, which corresponds to the number of times the task has been attempted, including the initial attempt. This is one more than the retry count.

        :return: The attempt number of the current task run.
        """

        return self.retry_count + 1

    @property
    def additional_metadata(self) -> JSONSerializableMapping | None:
        """
        The additional metadata sent with the current task run.

        :return: The additional metadata sent with the current task run, or None if no additional metadata was sent.
        """
        return self.action.additional_metadata

    @property
    def child_index(self) -> int | None:
        return self.action.child_workflow_index

    @property
    def child_key(self) -> str | None:
        return self.action.child_workflow_key

    @property
    def parent_workflow_run_id(self) -> str | None:
        """
        The parent workflow run id of the current task run, if it exists. This is useful for knowing which workflow run spawned this run as a child.

        :return: The parent workflow run id of the current task run, or None if it does not exist.
        """
        return self.action.parent_workflow_run_id

    @property
    def priority(self) -> int | None:
        """
        The priority that the current task was run with.

        :return: The priority of the current task run, or None if no priority was set.
        """
        return self.action.priority

    @property
    def workflow_id(self) -> str | None:
        """
        The id of the workflow that this task belongs to.

        :return: The id of the workflow that this task belongs to.
        """

        return self.action.workflow_id

    @property
    def workflow_version_id(self) -> str | None:
        """
        The id of the workflow version that this task belongs to.

        :return: The id of the workflow version that this task belongs to.
        """

        return self.action.workflow_version_id

    @property
    def task_run_errors(self) -> dict[str, str]:
        """
        A helper intended to be used in an on-failure step to retrieve the errors that occurred in upstream task runs.

        :return: A dictionary mapping task names to their error messages.
        """
        errors = self.data.step_run_errors

        if not errors:
            logger.error(
                "No step run errors found. `context.task_run_errors` is intended to be run in an on-failure step, and will only work on engine versions more recent than v0.53.10"
            )

        return errors

    def fetch_task_run_error(
        self,
        task: "Task[TWorkflowInput, R]",
    ) -> str | None:
        """
        A helper intended to be used in an on-failure step to retrieve the error that occurred in a specific upstream task run.

        :param task: The task whose error you want to retrieve.
        :return: The error message of the task run, or None if no error occurred.
        """
        errors = self.data.step_run_errors

        return errors.get(task.name)


class DurableContext(Context):
    async def aio_wait_for(
        self,
        signal_key: str,
        *conditions: SleepCondition | UserEventCondition | OrGroup,
    ) -> dict[str, Any]:
        """
        Durably wait for either a sleep or an event.

        :param signal_key: The key to use for the durable event. This is used to identify the event in the Hatchet API.
        :param *conditions: The conditions to wait for. Can be a SleepCondition or UserEventCondition.

        :return: A dictionary containing the results of the wait.
        :raises ValueError: If the durable event listener is not available.
        """
        if self.durable_event_listener is None:
            raise ValueError("Durable event listener is not available")

        task_id = self.step_run_id

        request = RegisterDurableEventRequest(
            task_id=task_id,
            signal_key=signal_key,
            conditions=flatten_conditions(list(conditions)),
            config=self.runs_client.client_config,
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
