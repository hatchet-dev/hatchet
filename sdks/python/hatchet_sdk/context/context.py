import asyncio
import json
from datetime import timedelta
from typing import TYPE_CHECKING, Any, cast
from warnings import warn

from hatchet_sdk.cancellation import CancellationToken
from hatchet_sdk.clients.admin import AdminClient, TriggerWorkflowOptions
from hatchet_sdk.clients.dispatcher.dispatcher import (  # type: ignore[attr-defined]
    Action,
    DispatcherClient,
)
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.listeners.durable_event_listener import (
    DurableEventListener,
)
from hatchet_sdk.conditions import (
    OrGroup,
    SleepCondition,
    UserEventCondition,
    build_conditions_proto,
    flatten_conditions,
)
from hatchet_sdk.context.worker_context import WorkerContext
from hatchet_sdk.contracts.v1.dispatcher_pb2 import DurableTaskEventKind
from hatchet_sdk.exceptions import CancellationReason, TaskRunError
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.types import EmptyModel, R, TWorkflowInput
from hatchet_sdk.utils.cancellation import await_with_cancellation
from hatchet_sdk.utils.timedelta_to_expression import Duration, timedelta_to_expr
from hatchet_sdk.utils.typing import JSONSerializableMapping, LogLevel
from hatchet_sdk.worker.durable_eviction.instrumentation import (
    aio_durable_eviction_wait,
)
from hatchet_sdk.worker.runner.utils.capture_logs import AsyncLogSender, LogRecord

if TYPE_CHECKING:
    from hatchet_sdk.runnables.task import Task
    from hatchet_sdk.runnables.workflow import BaseWorkflow


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
        max_attempts: int,
        task_name: str,
        workflow_name: str,
    ):
        self.worker = worker

        self.data = action.action_payload

        self.action = action

        self.step_run_id = action.step_run_id
        self.cancellation_token = CancellationToken()
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
        self._max_attempts = max_attempts
        self._workflow_name = workflow_name
        self._task_name = task_name

    @property
    def exit_flag(self) -> bool:
        """
        Check if the cancellation flag has been set.

        This property is maintained for backwards compatibility.
        Use `cancellation_token.is_cancelled` for new code.

        :return: True if the task has been cancelled, False otherwise.
        """
        return self.cancellation_token.is_cancelled

    @exit_flag.setter
    def exit_flag(self, value: bool) -> None:
        """
        Set the cancellation flag.

        This setter is maintained for backwards compatibility.
        Setting to True will trigger the cancellation token.

        :param value: True to trigger cancellation, False is a no-op.
        """
        if value:
            self.cancellation_token.cancel(CancellationReason.USER_REQUESTED)

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
        from hatchet_sdk.serde import HATCHET_PYDANTIC_SENTINEL

        if self.was_skipped(task):
            raise ValueError(f"{task.name} was skipped")

        try:
            parent_step_data = cast(R, self.data.parents[task.name])
        except KeyError as e:
            raise ValueError(f"Step output for '{task.name}' not found") from e

        return cast(
            R,
            task.validators.step_output.validate_python(
                parent_step_data, context=HATCHET_PYDANTIC_SENTINEL
            ),
        )

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

    def _set_cancellation_flag(
        self, reason: CancellationReason = CancellationReason.WORKFLOW_CANCELLED
    ) -> None:
        """
        Internal method to trigger cancellation.

        This triggers the cancellation token, which will:
        - Signal all waiters (async and sync)
        - Set the exit_flag property to True
        - Allow child workflow cancellation

        Args:
            reason: The reason for cancellation.
        """
        self.cancellation_token.cancel(reason)

    def cancel(self) -> None:
        """
        Cancel the current task run. This will call the Hatchet API to cancel the step run and set the exit flag to True.

        :return: None
        """
        self.runs_client.cancel(self.step_run_id)
        self._set_cancellation_flag(CancellationReason.USER_REQUESTED)

    async def aio_cancel(self) -> None:
        """
        Cancel the current task run. This will call the Hatchet API to cancel the step run and set the exit flag to True.

        :return: None
        """
        await self.runs_client.aio_cancel(self.step_run_id)
        self._set_cancellation_flag(CancellationReason.USER_REQUESTED)

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
        self.log_sender.publish(
            LogRecord(
                message=line,
                step_run_id=self.step_run_id,
                level=LogLevel.INFO,
                task_retry_count=self.retry_count,
            )
        )

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
        except Exception:
            logger.exception("error putting stream event")

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
        except Exception:
            logger.exception("error refreshing timeout")

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
    def max_attempts(self) -> int:
        """
        The maximum number of attempts allowed for the current task run, computed as the number of retries plus one.

        :return: The maximum number of attempts allowed for the current task run.
        """

        return self._max_attempts

    @property
    def additional_metadata(self) -> JSONSerializableMapping:
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
                "no step run errors found. `context.task_run_errors` is intended to be run in an on-failure step, and will only work on engine versions more recent than v0.53.10"
            )

        return errors

    @property
    def workflow_name(self) -> str:
        return self._workflow_name

    @property
    def task_name(self) -> str:
        return self._task_name

    def fetch_task_run_error(
        self,
        task: "Task[TWorkflowInput, R]",
    ) -> str | None:
        """
        **DEPRECATED**: Use `get_task_run_error` instead.

        A helper intended to be used in an on-failure step to retrieve the error that occurred in a specific upstream task run.

        :param task: The task whose error you want to retrieve.
        :return: The error message of the task run, or None if no error occurred.
        """
        warn(
            "`fetch_task_run_error` is deprecated. Use `get_task_run_error` instead.",
            DeprecationWarning,
            stacklevel=2,
        )
        errors = self.data.step_run_errors

        return errors.get(task.name)

    def get_task_run_error(
        self,
        task: "Task[TWorkflowInput, R]",
    ) -> TaskRunError | None:
        """
        A helper intended to be used in an on-failure step to retrieve the error that occurred in a specific upstream task run.

        :param task: The task whose error you want to retrieve.
        :return: The error message of the task run, or None if no error occurred.
        """
        errors = self.data.step_run_errors

        error = errors.get(task.name)

        if not error:
            return None

        return TaskRunError.deserialize(error)


class DurableContext(Context):
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
        max_attempts: int,
        task_name: str,
        workflow_name: str,
    ):
        super().__init__(
            action,
            dispatcher_client,
            admin_client,
            event_client,
            durable_event_listener,
            worker,
            runs_client,
            lifespan_context,
            log_sender,
            max_attempts,
            task_name,
            workflow_name,
        )

        self._wait_index = 0

    @property
    def wait_index(self) -> int:
        return self._wait_index

    def _increment_wait_index(self) -> int:
        index = self._wait_index
        self._wait_index += 1

        return index

    ## todo: instrumentor for this
    async def aio_wait_for(
        self,
        signal_key: str,
        *conditions: SleepCondition | UserEventCondition | OrGroup,
    ) -> dict[str, Any]:
        """
        Durably wait for either a sleep or an event.

        This method respects the context's cancellation token. If the task is cancelled
        while waiting, an asyncio.CancelledError will be raised.

        :param signal_key: The key to use for the durable event. This is used to identify the event in the Hatchet API.
        :param \\*conditions: The conditions to wait for. Can be a SleepCondition or UserEventCondition.

        :return: A dictionary containing the results of the wait.
        :raises ValueError: If the durable task client is not available.
        """
        if self.durable_event_listener is None:
            raise ValueError("Durable task client is not available")

        from hatchet_sdk.contracts.v1.dispatcher_pb2 import DurableTaskEventKind

        await self._ensure_stream_started()

        flat_conditions = flatten_conditions(list(conditions))
        conditions_proto = build_conditions_proto(
            flat_conditions, self.runs_client.client_config
        )
        invocation_count = self.attempt_number

        ack = await self.durable_event_listener.send_event(
            durable_task_external_id=self.step_run_id,
            ## todo: figure out how to store this invocation count properly
            invocation_count=invocation_count,
            kind=DurableTaskEventKind.DURABLE_TASK_TRIGGER_KIND_WAIT_FOR,
            payload=None,
            wait_for_conditions=conditions_proto,
        )
        node_id = ack.node_id

        async with aio_durable_eviction_wait(
            "durable_event", f"{self.step_run_id}:{signal_key}"
        ):
            result = await await_with_cancellation(
                self.durable_event_listener.wait_for_callback(
                    durable_task_external_id=self.step_run_id,
                    node_id=node_id,
                ),
                self.cancellation_token,
            )

        return result.payload or {}

    async def aio_sleep_for(self, duration: Duration) -> dict[str, Any]:
        """
        Lightweight wrapper for durable sleep. Allows for shorthand usage of `ctx.aio_wait_for` when specifying a sleep condition.

        This method respects the context's cancellation token. If the task is cancelled
        while sleeping, an asyncio.CancelledError will be raised.

        For more complicated conditions, use `ctx.aio_wait_for` directly.

        :param duration: The duration to sleep for.
        :return: A dictionary containing the results of the wait.
        """
        wait_index = self._increment_wait_index()

        return await self.aio_wait_for(
            f"sleep:{timedelta_to_expr(duration)}-{wait_index}",
            SleepCondition(duration=duration),
        )

    ## todo: instrumentor for this
    async def _spawn_child(
        self,
        workflow: "BaseWorkflow[TWorkflowInput]",
        input: TWorkflowInput = cast(Any, EmptyModel()),
        options: "TriggerWorkflowOptions" | None = None,
    ) -> dict[str, Any]:
        if self.durable_event_listener is None:
            raise ValueError("Durable task client is not available")

        await self._ensure_stream_started()

        ack = await self.durable_event_listener.send_event(
            durable_task_external_id=self.step_run_id,
            invocation_count=self.attempt_number,
            kind=DurableTaskEventKind.DURABLE_TASK_TRIGGER_KIND_RUN,
            payload=workflow._serialize_input(input),
            workflow_name=workflow.config.name,
            trigger_workflow_opts=options,
        )

        node_id = ack.node_id

        result = await self.durable_event_listener.wait_for_callback(
            durable_task_external_id=self.step_run_id,
            node_id=node_id,
        )

        return result.payload or {}

    async def _ensure_stream_started(self) -> None:
        if self.durable_event_listener is None:
            raise ValueError("Durable task client is not available")

        await self.durable_event_listener.ensure_started(self.action.worker_id)
