from importlib.metadata import version
from typing import Any, Callable, Collection, Coroutine

try:
    from opentelemetry.context import Context
    from opentelemetry.instrumentation.instrumentor import (  # type: ignore[attr-defined]
        BaseInstrumentor,
    )
    from opentelemetry.instrumentation.utils import unwrap
    from opentelemetry.metrics import MeterProvider, NoOpMeterProvider, get_meter
    from opentelemetry.trace import (
        NoOpTracerProvider,
        StatusCode,
        TracerProvider,
        get_tracer,
        get_tracer_provider,
    )
    from opentelemetry.trace.propagation.tracecontext import (
        TraceContextTextMapPropagator,
    )
    from wrapt import wrap_function_wrapper  # type: ignore[import-untyped]
except (RuntimeError, ImportError, ModuleNotFoundError):
    raise ModuleNotFoundError(
        "To use the HatchetInstrumentor, you must install Hatchet's `otel` extra using (e.g.) `pip install hatchet-sdk[otel]`"
    )

import hatchet_sdk
from hatchet_sdk.contracts.events_pb2 import Event
from hatchet_sdk.v0.clients.admin import (
    AdminClient,
    TriggerWorkflowOptions,
    WorkflowRunDict,
)
from hatchet_sdk.v0.clients.dispatcher.action_listener import Action
from hatchet_sdk.v0.clients.events import (
    BulkPushEventWithMetadata,
    EventClient,
    PushEventOptions,
)
from hatchet_sdk.v0.worker.runner.runner import Runner
from hatchet_sdk.v0.workflow_run import WorkflowRunRef

hatchet_sdk_version = version("hatchet-sdk")

InstrumentKwargs = TracerProvider | MeterProvider | None

OTEL_TRACEPARENT_KEY = "traceparent"


def create_traceparent() -> str | None:
    """
    Creates and returns a W3C traceparent header value using OpenTelemetry's context propagation.

    The traceparent header is used to propagate context information across service boundaries
    in distributed tracing systems. It follows the W3C Trace Context specification.

    :returns: A W3C-formatted traceparent header value if successful, None if the context
                    injection fails or no active span exists.\n
                    Example: `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01`
    """

    carrier: dict[str, str] = {}
    TraceContextTextMapPropagator().inject(carrier)

    return carrier.get("traceparent")


def parse_carrier_from_metadata(metadata: dict[str, str] | None) -> Context | None:
    """
    Parses OpenTelemetry trace context from a metadata dictionary.

    Extracts the trace context from metadata using the W3C Trace Context format,
    specifically looking for the `traceparent` header.

    :param metadata: A dictionary containing metadata key-value pairs,
                     potentially including the `traceparent` header. Can be None.

    :returns: The extracted OpenTelemetry Context object if a valid `traceparent`
              is found in the metadata, otherwise None.

    :Example:

    >>> metadata = {"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"}
    >>> context = parse_carrier_from_metadata(metadata)
    """

    if not metadata:
        return None

    traceparent = metadata.get(OTEL_TRACEPARENT_KEY)

    if not traceparent:
        return None

    return TraceContextTextMapPropagator().extract({OTEL_TRACEPARENT_KEY: traceparent})


def inject_traceparent_into_metadata(
    metadata: dict[str, str], traceparent: str | None = None
) -> dict[str, str]:
    """
    Injects OpenTelemetry `traceparent` into a metadata dictionary.

    Takes a metadata dictionary and an optional `traceparent` string,
    returning a new metadata dictionary with the `traceparent` added under the
    `OTEL_TRACEPARENT_KEY`. If no `traceparent` is provided, it attempts to create one.

    :param metadata: The metadata dictionary to inject the `traceparent` into.

    :param traceparent: The `traceparent` string to inject. If None, attempts to use
                        the current span.

    :returns: A new metadata dictionary containing the original metadata plus
              the injected `traceparent`, if one was available or could be created.

    :Example:

    >>> metadata = {"key": "value"}
    >>> new_metadata = inject_traceparent(metadata, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
    >>> print(new_metadata)
    {"key": "value", "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}
    """

    if not traceparent:
        traceparent = create_traceparent()

    if not traceparent:
        return metadata

    return {
        **metadata,
        OTEL_TRACEPARENT_KEY: traceparent,
    }


class HatchetInstrumentor(BaseInstrumentor):  # type: ignore[misc]
    def __init__(
        self,
        tracer_provider: TracerProvider | None = None,
        meter_provider: MeterProvider | None = None,
    ):
        """
        Hatchet OpenTelemetry instrumentor.

        The instrumentor provides an OpenTelemetry integration for Hatchet by setting up
        tracing and metrics collection.

        :param tracer_provider: TracerProvider | None: The OpenTelemetry TracerProvider to use.
                If not provided, the global tracer provider will be used.
        :param meter_provider: MeterProvider | None: The OpenTelemetry MeterProvider to use.
                If not provided, a no-op meter provider will be used.
        """

        self.tracer_provider = tracer_provider or get_tracer_provider()
        self.meter_provider = meter_provider or NoOpMeterProvider()

        super().__init__()

    def instrumentation_dependencies(self) -> Collection[str]:
        return tuple()

    def _instrument(self, **kwargs: InstrumentKwargs) -> None:
        self._tracer = get_tracer(__name__, hatchet_sdk_version, self.tracer_provider)
        self._meter = get_meter(__name__, hatchet_sdk_version, self.meter_provider)

        wrap_function_wrapper(
            hatchet_sdk,
            "worker.runner.runner.Runner.handle_start_step_run",
            self._wrap_handle_start_step_run,
        )
        wrap_function_wrapper(
            hatchet_sdk,
            "worker.runner.runner.Runner.handle_start_group_key_run",
            self._wrap_handle_get_group_key_run,
        )
        wrap_function_wrapper(
            hatchet_sdk,
            "worker.runner.runner.Runner.handle_cancel_action",
            self._wrap_handle_cancel_action,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.events.EventClient.push",
            self._wrap_push_event,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.events.EventClient.bulk_push",
            self._wrap_bulk_push_event,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.admin.AdminClient.run_workflow",
            self._wrap_run_workflow,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.admin.AdminClientAioImpl.run_workflow",
            self._wrap_async_run_workflow,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.admin.AdminClient.run_workflows",
            self._wrap_run_workflows,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.admin.AdminClientAioImpl.run_workflows",
            self._wrap_async_run_workflows,
        )

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_handle_start_step_run(
        self,
        wrapped: Callable[[Action], Coroutine[None, None, Exception | None]],
        instance: Runner,
        args: tuple[Action],
        kwargs: Any,
    ) -> Exception | None:
        action = args[0]
        traceparent = parse_carrier_from_metadata(action.additional_metadata)

        with self._tracer.start_as_current_span(
            "hatchet.start_step_run",
            attributes=action.otel_attributes,
            context=traceparent,
        ) as span:
            result = await wrapped(*args, **kwargs)

            if isinstance(result, Exception):
                span.set_status(StatusCode.ERROR, str(result))

            return result

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_handle_get_group_key_run(
        self,
        wrapped: Callable[[Action], Coroutine[None, None, Exception | None]],
        instance: Runner,
        args: tuple[Action],
        kwargs: Any,
    ) -> Exception | None:
        action = args[0]

        with self._tracer.start_as_current_span(
            "hatchet.get_group_key_run",
            attributes=action.otel_attributes,
        ) as span:
            result = await wrapped(*args, **kwargs)

            if isinstance(result, Exception):
                span.set_status(StatusCode.ERROR, str(result))

            return result

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_handle_cancel_action(
        self,
        wrapped: Callable[[str], Coroutine[None, None, Exception | None]],
        instance: Runner,
        args: tuple[str],
        kwargs: Any,
    ) -> Exception | None:
        step_run_id = args[0]

        with self._tracer.start_as_current_span(
            "hatchet.cancel_step_run",
            attributes={
                "hatchet.step_run_id": step_run_id,
            },
        ):
            return await wrapped(*args, **kwargs)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    def _wrap_push_event(
        self,
        wrapped: Callable[[str, dict[str, Any], PushEventOptions | None], Event],
        instance: EventClient,
        args: tuple[
            str,
            dict[str, Any],
            PushEventOptions | None,
        ],
        kwargs: dict[str, str | dict[str, Any] | PushEventOptions | None],
    ) -> Event:
        with self._tracer.start_as_current_span(
            "hatchet.push_event",
        ):
            return wrapped(*args, **kwargs)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    def _wrap_bulk_push_event(
        self,
        wrapped: Callable[
            [list[BulkPushEventWithMetadata], PushEventOptions | None], list[Event]
        ],
        instance: EventClient,
        args: tuple[
            list[BulkPushEventWithMetadata],
            PushEventOptions | None,
        ],
        kwargs: dict[str, list[BulkPushEventWithMetadata] | PushEventOptions | None],
    ) -> list[Event]:
        with self._tracer.start_as_current_span(
            "hatchet.bulk_push_event",
        ):
            return wrapped(*args, **kwargs)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    def _wrap_run_workflow(
        self,
        wrapped: Callable[[str, Any, TriggerWorkflowOptions | None], WorkflowRunRef],
        instance: AdminClient,
        args: tuple[str, Any, TriggerWorkflowOptions | None],
        kwargs: dict[str, str | Any | TriggerWorkflowOptions | None],
    ) -> WorkflowRunRef:
        with self._tracer.start_as_current_span(
            "hatchet.run_workflow",
        ):
            return wrapped(*args, **kwargs)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_async_run_workflow(
        self,
        wrapped: Callable[
            [str, Any, TriggerWorkflowOptions | None],
            Coroutine[None, None, WorkflowRunRef],
        ],
        instance: AdminClient,
        args: tuple[str, Any, TriggerWorkflowOptions | None],
        kwargs: dict[str, str | Any | TriggerWorkflowOptions | None],
    ) -> WorkflowRunRef:
        with self._tracer.start_as_current_span(
            "hatchet.run_workflow",
        ):
            return await wrapped(*args, **kwargs)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    def _wrap_run_workflows(
        self,
        wrapped: Callable[
            [list[WorkflowRunDict], TriggerWorkflowOptions | None], list[WorkflowRunRef]
        ],
        instance: AdminClient,
        args: tuple[
            list[WorkflowRunDict],
            TriggerWorkflowOptions | None,
        ],
        kwargs: dict[str, list[WorkflowRunDict] | TriggerWorkflowOptions | None],
    ) -> list[WorkflowRunRef]:
        with self._tracer.start_as_current_span(
            "hatchet.run_workflows",
        ):
            return wrapped(*args, **kwargs)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_async_run_workflows(
        self,
        wrapped: Callable[
            [list[WorkflowRunDict], TriggerWorkflowOptions | None],
            Coroutine[None, None, list[WorkflowRunRef]],
        ],
        instance: AdminClient,
        args: tuple[
            list[WorkflowRunDict],
            TriggerWorkflowOptions | None,
        ],
        kwargs: dict[str, list[WorkflowRunDict] | TriggerWorkflowOptions | None],
    ) -> list[WorkflowRunRef]:
        with self._tracer.start_as_current_span(
            "hatchet.run_workflows",
        ):
            return await wrapped(*args, **kwargs)

    def _uninstrument(self, **kwargs: InstrumentKwargs) -> None:
        self.tracer_provider = NoOpTracerProvider()
        self.meter_provider = NoOpMeterProvider()

        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_start_step_run")
        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_start_group_key_run")
        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_cancel_action")
        unwrap(hatchet_sdk, "clients.events.EventClient.push")
        unwrap(hatchet_sdk, "clients.events.EventClient.bulk_push")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.run_workflow")
        unwrap(hatchet_sdk, "clients.admin.AdminClientAioImpl.run_workflow")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.run_workflows")
        unwrap(hatchet_sdk, "clients.admin.AdminClientAioImpl.run_workflows")
