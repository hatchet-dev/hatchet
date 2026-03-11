from __future__ import annotations

import asyncio
import hashlib
import json
from collections.abc import Awaitable, Callable
from datetime import UTC, datetime, timedelta
from typing import TYPE_CHECKING, Any, ParamSpec, TypeVar, cast
from warnings import warn

from pydantic import BaseModel, TypeAdapter

from hatchet_sdk.clients.admin import (
    AdminClient,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.clients.dispatcher.dispatcher import (  # type: ignore[attr-defined]
    Action,
    DispatcherClient,
)
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.listeners.durable_event_listener import (
    DurableEventListener,
    DurableTaskEventMemoAck,
    DurableTaskEventRunAck,
    DurableTaskEventWaitForAck,
    MemoEvent,
    RunChildEvent,
    RunChildrenEvent,
    WaitForEvent,
)
from hatchet_sdk.clients.listeners.legacy.pre_eviction_durable_event_listener import (
    PreEvictionDurableEventListener,
)
from hatchet_sdk.conditions import (
    OrGroup,
    SleepCondition,
    UserEventCondition,
    build_conditions_proto,
    flatten_conditions,
)
from hatchet_sdk.context.pre_eviction import aio_wait_for_pre_eviction
from hatchet_sdk.context.worker_context import WorkerContext
from hatchet_sdk.deprecated.deprecation import semver_less_than
from hatchet_sdk.engine_version import MinEngineVersion
from hatchet_sdk.exceptions import TaskRunError
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.types import (
    R,
    TWorkflowInput,
    ValidTaskReturnType,
)
from hatchet_sdk.serde import HATCHET_PYDANTIC_SENTINEL
from hatchet_sdk.utils.timedelta_to_expression import (
    Duration,
    expr_to_timedelta,
    timedelta_to_expr,
)
from hatchet_sdk.utils.typing import JSONSerializableMapping, LogLevel
from hatchet_sdk.worker.durable_eviction.instrumentation import (
    aio_durable_eviction_wait,
)
from hatchet_sdk.worker.durable_eviction.manager import DurableEvictionManager
from hatchet_sdk.worker.runner.utils.capture_logs import AsyncLogSender, LogRecord

PMemo = ParamSpec("PMemo")
TMemo = TypeVar("TMemo", bound=ValidTaskReturnType)

if TYPE_CHECKING:
    from hatchet_sdk.runnables.task import Task


class Event(BaseModel):
    id: str
    tenant_id: str
    key: str
    payload: JSONSerializableMapping
    seen_at: datetime
    additional_metadata: JSONSerializableMapping | None
    scope: str | None


class SleepResult(BaseModel):
    duration: timedelta


def _compute_memo_key(task_run_external_id: str, *args: Any, **kwargs: Any) -> bytes:
    h = hashlib.sha256()
    h.update(task_run_external_id.encode())
    h.update(json.dumps(args, default=str, sort_keys=True).encode())
    h.update(json.dumps(kwargs, default=str, sort_keys=True).encode())
    return h.digest()


class Context:
    def __init__(
        self,
        action: Action,
        dispatcher_client: DispatcherClient,
        admin_client: AdminClient,
        event_client: EventClient,
        durable_event_listener: (
            DurableEventListener | PreEvictionDurableEventListener | None
        ),
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
        self._max_attempts = max_attempts
        self._workflow_name = workflow_name
        self._task_name = task_name

    def _increment_stream_index(self) -> int:
        index = self.stream_index
        self.stream_index += 1

        return index

    def was_skipped(self, task: Task[TWorkflowInput, R]) -> bool:
        """
        Check if a given task was skipped. You can read about skipping in [the docs](https://docs.hatchet.run/home/conditional-workflows#skip_if).

        :param task: The task to check the status of (skipped or not).
        :return: True if the task was skipped, False otherwise.
        """
        return self.data.parents.get(task.name, {}).get("skipped", False) is True

    @property
    def trigger_data(self) -> JSONSerializableMapping:
        return self.data.triggers

    def task_output(self, task: Task[TWorkflowInput, R]) -> R:
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

    def aio_task_output(self, task: Task[TWorkflowInput, R]) -> R:
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

    @property
    def worker_id(self) -> str:
        return self.action.worker_id

    def fetch_task_run_error(
        self,
        task: Task[TWorkflowInput, R],
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
        task: Task[TWorkflowInput, R],
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
        durable_event_listener: (
            DurableEventListener | PreEvictionDurableEventListener | None
        ),
        worker: WorkerContext,
        runs_client: RunsClient,
        lifespan_context: Any | None,
        log_sender: AsyncLogSender,
        max_attempts: int,
        task_name: str,
        workflow_name: str,
        durable_eviction_manager: DurableEvictionManager | None = None,
        engine_version: str | None = None,
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
        self._durable_eviction_manager = durable_eviction_manager
        self._engine_version = engine_version

    @property
    def _durable_listener(self) -> DurableEventListener:
        if self.durable_event_listener is None:
            raise ValueError("Durable task client is not available")

        if not isinstance(self.durable_event_listener, DurableEventListener):
            raise TypeError(
                "Expected DurableEventListener, got "
                f"{type(self.durable_event_listener).__name__}"
            )
        return self.durable_event_listener

    @property
    def _supports_durable_eviction(self) -> bool:
        if not self._engine_version:
            return False
        return not semver_less_than(
            self._engine_version, MinEngineVersion.DURABLE_EVICTION
        )

    @property
    def wait_index(self) -> int:
        return self._wait_index

    def _increment_wait_index(self) -> int:
        index = self._wait_index
        self._wait_index += 1

        return index

    ## IMPORTANT: This method is instrumented by HatchetInstrumentor._wrap_aio_wait_for.
    ## Keep the signature in sync with the instrumentor wrapper.
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

        :raises ValueError: If the durable task client is not available.
        :raises TypeError: If the durable event listener is not of type DurableEventListener or PreEvictionDurableEventListener.
        """
        if self.durable_event_listener is None:
            raise ValueError("Durable task client is not available")

        if not self._supports_durable_eviction:
            return await aio_wait_for_pre_eviction(self, signal_key, *conditions)

        listener = self._durable_listener

        await self._ensure_stream_started()

        flat_conditions = flatten_conditions(list(conditions))
        conditions_proto = build_conditions_proto(
            flat_conditions, self.runs_client.client_config
        )
        ack = await listener.send_event(
            durable_task_external_id=self.step_run_id,
            invocation_count=self.invocation_count,
            event=WaitForEvent(wait_for_conditions=conditions_proto),
        )

        if not isinstance(ack, DurableTaskEventWaitForAck):
            raise TypeError(f"Expected wait-for ack, got {type(ack).__name__}")

        node_id = ack.node_id
        branch_id = ack.branch_id

        async with aio_durable_eviction_wait(
            wait_kind="wait_for",
            resource_id=signal_key,
            action_key=self.action.key,
            eviction_manager=self._durable_eviction_manager,
        ):
            result = await listener.wait_for_callback(
                durable_task_external_id=self.step_run_id,
                node_id=node_id,
                branch_id=branch_id,
                invocation_count=self.invocation_count,
            )

        return result.payload or {}

    async def aio_sleep_for(self, duration: Duration) -> SleepResult:
        """
        Lightweight wrapper for durable sleep. Allows for shorthand usage of `ctx.aio_wait_for` when specifying a sleep condition.

        For more complicated conditions, use `ctx.aio_wait_for` directly.
        """

        wait_index = self._increment_wait_index()

        res = await self.aio_wait_for(
            f"sleep:{timedelta_to_expr(duration)}-{wait_index}",
            SleepCondition(duration=duration),
        )

        ## lots of implicit use of engine semantics / internal logic here.
        ## the engine returns an object like this:
        ## {"CREATE": {"signal_key_1": [{"id": ...}]}}
        ## since we have a single match we're looking for, we know that
        ## the list of matches will only have one item, so we can extract and parse it
        matches: dict[str, list[dict[str, Any]]] = res.get("CREATE", {})
        _, raw_matches = next(iter(matches.items()))
        sleep = raw_matches[0]

        return SleepResult(
            duration=expr_to_timedelta(
                sleep.get("sleep_duration", timedelta_to_expr(duration))
            )
        )

    async def aio_wait_for_event(
        self, key: str, expression: str | None = None
    ) -> Event:
        """
        Lightweight wrapper for waiting for a user event. Allows for shorthand usage of `ctx.aio_wait_for` when specifying a user event condition.

        For more complicated conditions, use `ctx.aio_wait_for` directly.
        """

        wait_index = self._increment_wait_index()

        result = await self.aio_wait_for(
            f"event:{key}-{wait_index}",
            UserEventCondition(event_key=key, expression=expression),
        )

        ## lots of implicit use of engine semantics / internal logic here.
        ## the engine returns an object like this:
        ## {"CREATE": {"signal_key_1": [{"id": ...}]}}
        ## since we have a single match we're looking for, we know that
        ## the list of matches will only have one item, so we can extract and parse it
        matches: dict[str, list[dict[str, Any]]] = result.get("CREATE", {})
        _, raw_matches = next(iter(matches.items()))
        event = raw_matches[0]

        return Event(
            id=event.get("id", ""),
            tenant_id=self.action.tenant_id,
            key=event.get("key", key),
            payload=event.get("data", {}),
            seen_at=datetime.fromisoformat(
                event.get("seen_at", datetime.now(UTC).isoformat())
            ),
            additional_metadata=event.get("additional_metadata"),
            scope=event.get("scope"),
        )

    ## IMPORTANT: This method is instrumented by HatchetInstrumentor._wrap_spawn_children_no_wait.
    ## Keep the signature in sync with the instrumentor wrapper.
    async def _spawn_children_no_wait(
        self,
        configs: list[WorkflowRunTriggerConfig],
    ) -> list[tuple[int, int, str]]:
        listener = self._durable_listener

        await self._ensure_stream_started()

        ack = await listener.send_event(
            durable_task_external_id=self.step_run_id,
            invocation_count=self.invocation_count,
            event=RunChildrenEvent(
                children=[
                    RunChildEvent(
                        workflow_name=c.workflow_name,
                        input=c.input,
                        trigger_workflow_opts=c.options,
                    )
                    for c in configs
                ]
            ),
        )

        if not isinstance(ack, DurableTaskEventRunAck):
            raise TypeError(f"Expected run ack, got {type(ack).__name__}")

        return [
            (entry.node_id, entry.branch_id, configs[i].workflow_name)
            for i, entry in enumerate(ack.run_entries)
        ]

    async def _aio_result_for_spawned_child(
        self,
        node_id: int,
        branch_id: int,
        workflow_name: str,
    ) -> dict[str, Any]:
        listener = self._durable_listener

        async with aio_durable_eviction_wait(
            wait_kind="spawn_child",
            resource_id=workflow_name,
            action_key=self.action.key,
            eviction_manager=self._durable_eviction_manager,
        ):
            result = await listener.wait_for_callback(
                durable_task_external_id=self.step_run_id,
                node_id=node_id,
                branch_id=branch_id,
                invocation_count=self.invocation_count,
            )

        return result.payload or {}

    async def _ensure_stream_started(self) -> None:
        if not isinstance(self.durable_event_listener, DurableEventListener):
            raise ValueError("Durable task client is not available")

        await self.durable_event_listener.ensure_started(self.action.worker_id)

    @property
    def invocation_count(self) -> int:
        return self.action.durable_task_invocation_count or 1

    ## IMPORTANT: This method is instrumented by HatchetInstrumentor._wrap_aio_memo.
    ## Keep the signature in sync with the instrumentor wrapper.
    async def _aio_memo(
        self,
        fn: Callable[PMemo, Awaitable[TMemo]],
        result_validator: type[TMemo],
        /,
        *args: PMemo.args,
        **kwargs: PMemo.kwargs,
    ) -> TMemo:
        """
        Memoize a function by storing its result in durable storage. This is useful for caching the results of expensive computations that you don't want to repeat on every workflow replay without needing to spawn a child workflow or set up an external cache. The function signature is intended to behave similarly to `asyncio.to_thread` or other similar uses of partially applied functions, where you pass in the function and its arguments separately.

        Note that memoization is performed at the _task run_ level, meaning you cannot cache across tasks (whether they're part of the same workflow or otherwise).

        :param fn: The function to compute the value to be memoized. This should be an async function that returns the value to be memoized.
        :param result_validator: The type of the result to be memoized. This is used for validating the result when it's retrieved from durable storage and for properly serializing the result of the function call. This is required and generally we recommend using either a Pydantic model, a dataclass, or a TypedDict, but you can also use `dict` as an escape hatch.
        :param *args: The arguments to pass to the function when computing the value to be memoized. These are used for computing the memoization key, so that different arguments will result in different cached values.
        :param **kwargs: The keyword arguments to pass to the function when computing the value to be memoized. These are used for computing the memoization key, so that different keyword arguments will result in different cached values.

        :return: The memoized value, either retrieved from durable storage or computed by calling the function.

        :raises TypeError: If the durable event listener is not of type DurableEventListener or PreEvictionDurableEventListener.
        """
        if not self._supports_durable_eviction:
            logger.warning(
                "Engine does not support memoization (requires >= %s). "
                "aio_memo will execute the function but results will not be "
                "persisted across replays. Upgrade your engine to enable durable memoization.",
                MinEngineVersion.DURABLE_EVICTION,
            )
            return await fn(*args, **kwargs)

        listener = self._durable_listener

        run_external_id = self.step_run_id
        adapter = TypeAdapter(result_validator)

        key = _compute_memo_key(self.step_run_id, *args, **kwargs)

        ack = await listener.send_event(
            durable_task_external_id=run_external_id,
            invocation_count=self.invocation_count,
            event=MemoEvent(memo_key=key, result=None),
        )

        if not isinstance(ack, DurableTaskEventMemoAck):
            raise TypeError(f"Expected memo ack, got {type(ack).__name__}")

        if ack.memo_already_existed and ack.memo_result_payload is None:
            logger.warning(
                "memo key found in durable storage but no data was returned. rerunning the function to recompute the value. "
            )

        if ack.memo_already_existed and ack.memo_result_payload is not None:
            serialized_result = ack.memo_result_payload
            result = adapter.validate_json(
                serialized_result, context=HATCHET_PYDANTIC_SENTINEL
            )
        else:
            result = await fn(*args, **kwargs)
            serialized_result = adapter.dump_json(
                result, context=HATCHET_PYDANTIC_SENTINEL
            )

            await self._ensure_stream_started()

            await listener.send_memo_completed_notification(
                durable_task_external_id=run_external_id,
                node_id=ack.node_id,
                branch_id=ack.branch_id,
                invocation_count=self.invocation_count,
                memo_result_payload=serialized_result,
                memo_key=key,
            )

        return result
