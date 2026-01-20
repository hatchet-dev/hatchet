import asyncio
from collections.abc import AsyncIterator, Callable
from contextlib import (
    AbstractAsyncContextManager,
    AbstractContextManager,
    asynccontextmanager,
)
from dataclasses import asdict, is_dataclass
from inspect import Parameter, iscoroutinefunction, signature
from typing import (
    TYPE_CHECKING,
    Annotated,
    Any,
    Concatenate,
    Generic,
    ParamSpec,
    Protocol,
    TypeGuard,
    TypeVar,
    cast,
    get_args,
    get_origin,
    get_type_hints,
)

from pydantic import BaseModel, ConfigDict, TypeAdapter

from hatchet_sdk.conditions import (
    Action,
    Condition,
    OrGroup,
    ParentCondition,
    SleepCondition,
    UserEventCondition,
    flatten_conditions,
)
from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.context.worker_context import WorkerContext
from hatchet_sdk.contracts.v1.shared.condition_pb2 import TaskConditions
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    CreateTaskOpts,
    CreateTaskRateLimit,
    DesiredWorkerLabels,
)
from hatchet_sdk.exceptions import InvalidDependencyError
from hatchet_sdk.runnables.types import (
    ConcurrencyExpression,
    R,
    StepType,
    TaskIOValidator,
    TWorkflowInput,
    TWorkflowInput_contra,
    is_async_fn,
    is_sync_fn,
    normalize_validator,
)
from hatchet_sdk.serde import HATCHET_PYDANTIC_SENTINEL
from hatchet_sdk.utils.timedelta_to_expression import Duration, timedelta_to_expr
from hatchet_sdk.utils.typing import (
    AwaitableLike,
    CoroutineLike,
    JSONSerializableMapping,
)
from hatchet_sdk.worker.runner.utils.capture_logs import AsyncLogSender

if TYPE_CHECKING:
    from hatchet_sdk.runnables.workflow import Workflow

T = TypeVar("T")
T_co = TypeVar("T_co", covariant=True)
P = ParamSpec("P")


def is_async_context_manager(obj: Any) -> TypeGuard[AbstractAsyncContextManager[Any]]:
    """Type guard to check if an object is an async context manager."""
    return hasattr(obj, "__aenter__") and hasattr(obj, "__aexit__")


def is_sync_context_manager(obj: Any) -> TypeGuard[AbstractContextManager[Any]]:
    """Type guard to check if an object is a sync context manager."""
    return hasattr(obj, "__enter__") and hasattr(obj, "__exit__")


class DependencyFunc(Protocol[T_co, TWorkflowInput_contra]):
    def __call__(
        self, input: TWorkflowInput_contra, ctx: Context, *args: Any, **kwargs: Any
    ) -> (
        T_co
        | CoroutineLike[T_co]
        | AbstractContextManager[T_co]
        | AbstractAsyncContextManager[T_co]
    ): ...

    def __name__(self) -> str: ...


class Depends(Generic[T, TWorkflowInput]):
    def __init__(
        self,
        fn: DependencyFunc[T, TWorkflowInput],
    ) -> None:
        sig = signature(fn)
        params = list(sig.parameters.values())

        if len(params) < 2:
            raise InvalidDependencyError(
                f"Dependency function {fn.__name__} must have at least two parameters: input and ctx. "
                f"Additional parameters can be dependencies."
            )

        self.fn = fn


class DependencyToInject(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    name: str
    value: Any


class Task(Generic[TWorkflowInput, R]):
    def __init__(
        self,
        _fn: (
            Callable[Concatenate[TWorkflowInput, Context, P], R | CoroutineLike[R]]
            | Callable[Concatenate[TWorkflowInput, Context, P], AwaitableLike[R]]
            | (
                Callable[
                    Concatenate[TWorkflowInput, DurableContext, P], R | CoroutineLike[R]
                ]
                | Callable[
                    Concatenate[TWorkflowInput, DurableContext, P], AwaitableLike[R]
                ]
            )
        ),
        is_durable: bool,
        type: StepType,
        workflow: "Workflow[TWorkflowInput]",
        name: str,
        execution_timeout: Duration,
        schedule_timeout: Duration,
        parents: "list[Task[TWorkflowInput, Any]] | None",
        retries: int,
        rate_limits: list[CreateTaskRateLimit] | None,
        desired_worker_labels: dict[str, DesiredWorkerLabels] | None,
        backoff_factor: float | None,
        backoff_max_seconds: int | None,
        concurrency: int | list[ConcurrencyExpression] | None,
        wait_for: list[Condition | OrGroup] | None,
        skip_if: list[Condition | OrGroup] | None,
        cancel_if: list[Condition | OrGroup] | None,
    ) -> None:
        self.is_durable = is_durable

        self.fn = _fn
        self.is_async_function = is_async_fn(self.fn)  # type: ignore

        self.workflow = workflow

        self.type = type
        self.execution_timeout = execution_timeout
        self.schedule_timeout = schedule_timeout
        self.name = name
        self.parents = parents or []
        self.retries = retries
        self.rate_limits = rate_limits or []
        self.desired_worker_labels = desired_worker_labels or {}
        self.backoff_factor = backoff_factor
        self.backoff_max_seconds = backoff_max_seconds
        self.concurrency = concurrency or []

        self.wait_for = flatten_conditions(wait_for or [])
        self.skip_if = flatten_conditions(skip_if or [])
        self.cancel_if = flatten_conditions(cancel_if or [])

        return_type = get_type_hints(_fn).get("return")

        self.validators: TaskIOValidator = TaskIOValidator(
            workflow_input=workflow.config.input_validator,
            step_output=TypeAdapter(normalize_validator(return_type)),
        )

    async def _parse_maybe_cm_param(
        self,
        parsed: DependencyToInject,
        cms_to_exit: (
            list[AbstractAsyncContextManager[Any] | AbstractContextManager[Any]] | None
        ),
    ) -> tuple[
        Any, AbstractAsyncContextManager[Any] | AbstractContextManager[Any] | None
    ]:
        value = parsed.value
        to_exit: (
            AbstractAsyncContextManager[Any] | AbstractContextManager[Any] | None
        ) = None

        if is_async_context_manager(value):
            entered_value: Any = await value.__aenter__()

            if cms_to_exit is not None:
                to_exit = value

            return entered_value, to_exit

        if is_sync_context_manager(value):
            entered_value = await asyncio.to_thread(value.__enter__)

            if cms_to_exit is not None:
                to_exit = value

            return entered_value, to_exit

        return value, to_exit

    async def _resolve_function_dependencies(
        self,
        fn: Callable[..., Any],
        input: TWorkflowInput,
        ctx: Context | DurableContext,
        resolution_stack: set[str] | None = None,  # detect cycles
        cms_to_exit: (
            list[AbstractAsyncContextManager[Any] | AbstractContextManager[Any]] | None
        ) = None,
    ) -> dict[str, Any]:
        if resolution_stack is None:
            resolution_stack = set()

        fn_name = fn.__name__
        if fn_name in resolution_stack:
            stack_path = " -> ".join(resolution_stack)
            raise InvalidDependencyError(
                f"Circular dependency detected: {fn_name} is already being resolved. "
                f"Dependency chain: {stack_path} -> {fn_name}"
            )

        resolution_stack.add(fn_name)
        try:
            sig = signature(fn)
            params = list(sig.parameters.items())

            dependencies: dict[str, Any] = {}

            for name, param in params[2:]:  # first two params are input and ctx
                parsed = await self._parse_parameter(
                    name, param, input, ctx, resolution_stack, cms_to_exit
                )
                if parsed is not None:
                    value, to_exit = await self._parse_maybe_cm_param(
                        parsed, cms_to_exit
                    )

                    dependencies[parsed.name] = value
                    if to_exit is not None and cms_to_exit is not None:
                        cms_to_exit.append(to_exit)

            return dependencies
        finally:
            resolution_stack.discard(fn_name)

    async def _parse_parameter(
        self,
        name: str,
        param: Parameter,
        input: TWorkflowInput,
        ctx: Context | DurableContext,
        resolution_stack: set[str] | None = None,
        cms_to_exit: (
            list[AbstractAsyncContextManager[Any] | AbstractContextManager[Any]] | None
        ) = None,
    ) -> DependencyToInject | None:
        annotation = param.annotation

        if get_origin(annotation) is Annotated:
            args = get_args(annotation)

            if len(args) < 2:
                return None

            metadata = args[1:]

            for item in metadata:
                if isinstance(item, Depends):
                    deps = await self._resolve_function_dependencies(
                        item.fn, input, ctx, resolution_stack, cms_to_exit
                    )

                    if iscoroutinefunction(item.fn):
                        return DependencyToInject(
                            name=name, value=await item.fn(input, ctx, **deps)
                        )

                    return DependencyToInject(
                        name=name,
                        value=await asyncio.to_thread(item.fn, input, ctx, **deps),
                    )

        return None

    async def _unpack_dependencies(
        self, ctx: Context | DurableContext
    ) -> dict[str, Any]:
        sig = signature(self.fn)
        input = self.workflow._get_workflow_input(ctx)
        return {
            parsed.name: parsed.value
            for n, p in sig.parameters.items()
            if (parsed := await self._parse_parameter(n, p, input, ctx)) is not None
        }

    @asynccontextmanager
    async def _unpack_dependencies_with_cleanup(
        self, ctx: Context | DurableContext
    ) -> AsyncIterator[dict[str, Any]]:
        sig = signature(self.fn)
        input = self.workflow._get_workflow_input(ctx)

        dependencies: dict[str, Any] = {}
        cms_to_exit: list[
            AbstractAsyncContextManager[Any] | AbstractContextManager[Any]
        ] = []

        try:
            for n, p in sig.parameters.items():
                parsed = await self._parse_parameter(
                    n, p, input, ctx, None, cms_to_exit
                )
                if parsed is not None:
                    value, to_exit = await self._parse_maybe_cm_param(
                        parsed, cms_to_exit
                    )

                    dependencies[parsed.name] = value
                    if to_exit is not None:
                        cms_to_exit.append(to_exit)

            yield dependencies
        finally:
            for cm in reversed(cms_to_exit):
                if is_async_context_manager(cm):
                    await cm.__aexit__(None, None, None)
                elif is_sync_context_manager(cm):
                    await asyncio.to_thread(cm.__exit__, None, None, None)

    def call(
        self, ctx: Context | DurableContext, dependencies: dict[str, Any] | None = None
    ) -> R:
        if self.is_async_function:
            raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

        workflow_input = self.workflow._get_workflow_input(ctx)
        dependencies = dependencies or {}

        if is_sync_fn(self.fn):  # type: ignore
            return self.fn(workflow_input, cast(Context, ctx), **dependencies)  # type: ignore

        raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

    async def aio_call(
        self, ctx: Context | DurableContext, dependencies: dict[str, Any] | None = None
    ) -> R:
        if not self.is_async_function:
            raise TypeError(
                f"{self.name} is not an async function. Use `call` instead."
            )

        workflow_input = self.workflow._get_workflow_input(ctx)
        dependencies = dependencies or {}

        if is_async_fn(self.fn):  # type: ignore
            return await self.fn(workflow_input, cast(Context, ctx), **dependencies)  # type: ignore

        raise TypeError(f"{self.name} is not an async function. Use `call` instead.")

    def to_proto(self, service_name: str) -> CreateTaskOpts:
        if isinstance(self.concurrency, int):
            concurrency = [ConcurrencyExpression.from_int(self.concurrency)]
        else:
            concurrency = self.concurrency

        return CreateTaskOpts(
            readable_id=self.name,
            action=service_name + ":" + self.name,
            timeout=timedelta_to_expr(self.execution_timeout),
            inputs="{}",
            parents=[p.name for p in self.parents],
            retries=self.retries,
            rate_limits=self.rate_limits,
            worker_labels=self.desired_worker_labels,
            backoff_factor=self.backoff_factor,
            backoff_max_seconds=self.backoff_max_seconds,
            concurrency=[t.to_proto() for t in concurrency],
            conditions=self._conditions_to_proto(),
            schedule_timeout=timedelta_to_expr(self.schedule_timeout),
        )

    def _assign_action(self, condition: Condition, action: Action) -> Condition:
        condition.base.action = action

        return condition

    def _conditions_to_proto(self) -> TaskConditions:
        wait_for_conditions = [
            self._assign_action(w, Action.QUEUE) for w in self.wait_for
        ]

        cancel_if_conditions = [
            self._assign_action(c, Action.CANCEL) for c in self.cancel_if
        ]
        skip_if_conditions = [self._assign_action(s, Action.SKIP) for s in self.skip_if]

        conditions = wait_for_conditions + cancel_if_conditions + skip_if_conditions

        if len({c.base.readable_data_key for c in conditions}) != len(
            [c.base.readable_data_key for c in conditions]
        ):
            raise ValueError("Conditions must have unique readable data keys.")

        user_events = [
            c.to_proto(self.workflow.client.config)
            for c in conditions
            if isinstance(c, UserEventCondition)
        ]
        parent_overrides = [
            c.to_proto(self.workflow.client.config)
            for c in conditions
            if isinstance(c, ParentCondition)
        ]
        sleep_conditions = [
            c.to_proto(self.workflow.client.config)
            for c in conditions
            if isinstance(c, SleepCondition)
        ]

        return TaskConditions(
            parent_override_conditions=parent_overrides,
            sleep_conditions=sleep_conditions,
            user_event_conditions=user_events,
        )

    def _create_mock_context(
        self,
        input: TWorkflowInput | None,
        additional_metadata: JSONSerializableMapping | None = None,
        parent_outputs: dict[str, JSONSerializableMapping] | None = None,
        retry_count: int = 0,
        lifespan_context: Any = None,
    ) -> Context | DurableContext:
        from hatchet_sdk.runnables.action import Action, ActionPayload, ActionType

        additional_metadata = additional_metadata or {}
        parent_outputs = parent_outputs or {}
        serialized_input: dict[str, Any] = {}

        if is_dataclass(input):
            serialized_input = asdict(input)
        elif isinstance(input, BaseModel):
            serialized_input = input.model_dump(context=HATCHET_PYDANTIC_SENTINEL)

        action_payload = ActionPayload(input=serialized_input, parents=parent_outputs)

        action = Action(
            tenant_id=self.workflow.client.config.tenant_id,
            worker_id="mock-worker-id",
            workflow_run_id="mock-workflow-run-id",
            job_id="mock-job-id",
            job_name="mock-job-name",
            job_run_id="mock-job-run-id",
            step_id="mock-step-id",
            step_run_id="mock-step-run-id",
            action_id="mock:action",
            action_payload=action_payload,
            action_type=ActionType.START_STEP_RUN,
            retry_count=retry_count,
            additional_metadata=additional_metadata,
            child_workflow_index=None,
            child_workflow_key=None,
            parent_workflow_run_id=None,
            priority=1,
            workflow_version_id="mock-workflow-version-id",
            workflow_id="mock-workflow-id",
        )

        constructor = DurableContext if self.is_durable else Context

        return constructor(
            action=action,
            dispatcher_client=self.workflow.client._client.dispatcher,
            admin_client=self.workflow.client._client.admin,
            event_client=self.workflow.client._client.event,
            durable_event_listener=None,
            worker=WorkerContext(
                labels={}, client=self.workflow.client._client.dispatcher
            ),
            runs_client=self.workflow.client._client.runs,
            lifespan_context=lifespan_context,
            log_sender=AsyncLogSender(self.workflow.client._client.event),
            max_attempts=self.retries + 1,
        )

    def mock_run(
        self,
        input: TWorkflowInput | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        parent_outputs: dict[str, JSONSerializableMapping] | None = None,
        retry_count: int = 0,
        lifespan: Any = None,
        dependencies: dict[str, Any] | None = None,
    ) -> R:
        """
        Mimic the execution of a task. This method is intended to be used to unit test
        tasks without needing to interact with the Hatchet engine. Use `mock_run` for sync
        tasks and `aio_mock_run` for async tasks.

        :param input: The input to the task.
        :param additional_metadata: Additional metadata to attach to the task.
        :param parent_outputs: Outputs from parent tasks, if any. This is useful for mimicking DAG functionality. For instance, if you have a task `step_2` that has a `parent` which is `step_1`, you can pass `parent_outputs={"step_1": {"result": "Hello, world!"}}` to `step_2.mock_run()` to be able to access `ctx.task_output(step_1)` in `step_2`.
        :param retry_count: The number of times the task has been retried.
        :param lifespan: The lifespan to be used in the task, which is useful if one was set on the worker. This will allow you to access `ctx.lifespan` inside of your task.
        :param dependencies: Dependencies to be injected into the task. This is useful for tasks that have dependencies defined using `Depends`. **IMPORTANT**: You must pass the dependencies _directly_, **not** the `Depends` objects themselves. For example, if you have a task that has a dependency `config: Annotated[str, Depends(get_config)]`, you should pass `dependencies={"config": "config_value"}` to `aio_mock_run`.

        :return: The output of the task.
        :raises TypeError: If the task is an async function and `mock_run` is called, or if the task is a sync function and `aio_mock_run` is called.
        """

        if self.is_async_function:
            raise TypeError(
                f"{self.name} is not a sync function. Use `aio_mock_run` instead."
            )

        ctx = self._create_mock_context(
            input, additional_metadata, parent_outputs, retry_count, lifespan
        )

        return self.call(ctx, dependencies)

    async def aio_mock_run(
        self,
        input: TWorkflowInput | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        parent_outputs: dict[str, JSONSerializableMapping] | None = None,
        retry_count: int = 0,
        lifespan: Any = None,
        dependencies: dict[str, Any] | None = None,
    ) -> R:
        """
        Mimic the execution of a task. This method is intended to be used to unit test
        tasks without needing to interact with the Hatchet engine. Use `mock_run` for sync
        tasks and `aio_mock_run` for async tasks.

        :param input: The input to the task.
        :param additional_metadata: Additional metadata to attach to the task.
        :param parent_outputs: Outputs from parent tasks, if any. This is useful for mimicking DAG functionality. For instance, if you have a task `step_2` that has a `parent` which is `step_1`, you can pass `parent_outputs={"step_1": {"result": "Hello, world!"}}` to `step_2.mock_run()` to be able to access `ctx.task_output(step_1)` in `step_2`.
        :param retry_count: The number of times the task has been retried.
        :param lifespan: The lifespan to be used in the task, which is useful if one was set on the worker. This will allow you to access `ctx.lifespan` inside of your task.
        :param dependencies: Dependencies to be injected into the task. This is useful for tasks that have dependencies defined using `Depends`. **IMPORTANT**: You must pass the dependencies _directly_, **not** the `Depends` objects themselves. For example, if you have a task that has a dependency `config: Annotated[str, Depends(get_config)]`, you should pass `dependencies={"config": "config_value"}` to `aio_mock_run`.

        :return: The output of the task.
        :raises TypeError: If the task is an async function and `mock_run` is called, or if the task is a sync function and `aio_mock_run` is called.
        """

        if not self.is_async_function:
            raise TypeError(
                f"{self.name} is not an async function. Use `mock_run` instead."
            )

        ctx = self._create_mock_context(
            input,
            additional_metadata,
            parent_outputs,
            retry_count,
            lifespan,
        )

        return await self.aio_call(ctx, dependencies)
