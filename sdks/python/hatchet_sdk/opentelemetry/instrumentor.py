import json
from importlib.metadata import version
from typing import Any, Callable, Collection, Coroutine, cast

from hatchet_sdk.utils.typing import JSONSerializableMapping

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

import inspect

import hatchet_sdk
from hatchet_sdk import ClientConfig
from hatchet_sdk.clients.admin import (
    AdminClient,
    TriggerWorkflowOptions,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.clients.events import (
    BulkPushEventWithMetadata,
    EventClient,
    PushEventOptions,
)
from hatchet_sdk.contracts.events_pb2 import Event
from hatchet_sdk.runnables.action import Action
from hatchet_sdk.utils.opentelemetry import OTelAttribute
from hatchet_sdk.worker.runner.runner import Runner
from hatchet_sdk.workflow_run import WorkflowRunRef

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


def parse_carrier_from_metadata(
    metadata: JSONSerializableMapping | None,
) -> Context | None:
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
    """
    Hatchet OpenTelemetry instrumentor.

    The instrumentor provides an OpenTelemetry integration for Hatchet by setting up
    tracing and metrics collection.

    :param tracer_provider: TracerProvider | None: The OpenTelemetry TracerProvider to use.
            If not provided, the global tracer provider will be used.
    :param meter_provider: MeterProvider | None: The OpenTelemetry MeterProvider to use.
            If not provided, a no-op meter provider will be used.
    :param config: ClientConfig | None: The configuration for the Hatchet client. If not provided,
            a default configuration will be used.
    """

    def __init__(
        self,
        tracer_provider: TracerProvider | None = None,
        meter_provider: MeterProvider | None = None,
        config: ClientConfig | None = None,
    ):
        self.config = config or ClientConfig()

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
            "clients.admin.AdminClient.aio_run_workflow",
            self._wrap_async_run_workflow,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.admin.AdminClient.run_workflows",
            self._wrap_run_workflows,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "clients.admin.AdminClient.aio_run_workflows",
            self._wrap_async_run_workflows,
        )

    def extract_bound_args(
        self,
        wrapped_func: Callable[..., Any],
        args: tuple[Any, ...],
        kwargs: dict[str, Any],
    ) -> list[Any]:
        sig = inspect.signature(wrapped_func)

        bound_args = sig.bind(*args, **kwargs)
        bound_args.apply_defaults()

        return list(bound_args.arguments.values())

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_handle_start_step_run(
        self,
        wrapped: Callable[[Action], Coroutine[None, None, Exception | None]],
        instance: Runner,
        args: tuple[Action],
        kwargs: Any,
    ) -> Exception | None:
        params = self.extract_bound_args(wrapped, args, kwargs)

        action = cast(Action, params[0])

        traceparent = parse_carrier_from_metadata(action.additional_metadata)

        with self._tracer.start_as_current_span(
            "hatchet.start_step_run",
            attributes=action.get_otel_attributes(self.config),
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
            attributes=action.get_otel_attributes(self.config),
        ) as span:
            result = await wrapped(*args, **kwargs)

            if isinstance(result, Exception):
                span.set_status(StatusCode.ERROR, str(result))

            return result

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_handle_cancel_action(
        self,
        wrapped: Callable[[Action], Coroutine[None, None, Exception | None]],
        instance: Runner,
        args: tuple[Action],
        kwargs: Any,
    ) -> Exception | None:
        action = args[0]

        with self._tracer.start_as_current_span(
            "hatchet.cancel_step_run",
            attributes={
                "hatchet.step_run_id": action.step_run_id,
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
        params = self.extract_bound_args(wrapped, args, kwargs)

        event_key = cast(str, params[0])
        payload = cast(JSONSerializableMapping, params[1])
        options = cast(
            PushEventOptions,
            params[2] if len(params) > 2 else PushEventOptions(),
        )

        attributes = {
            OTelAttribute.EVENT_KEY: event_key,
            OTelAttribute.EVENT_PAYLOAD: json.dumps(payload, default=str),
            OTelAttribute.EVENT_ADDITIONAL_METADATA: json.dumps(
                options.additional_metadata, default=str
            ),
            OTelAttribute.EVENT_NAMESPACE: options.namespace,
            OTelAttribute.EVENT_PRIORITY: options.priority,
            OTelAttribute.EVENT_SCOPE: options.scope,
        }

        with self._tracer.start_as_current_span(
            "hatchet.push_event",
            attributes={
                f"hatchet.{k.value}": v
                for k, v in attributes.items()
                if v and k not in self.config.otel.excluded_attributes
            },
        ):
            options = PushEventOptions(
                **options.model_dump(exclude={"additional_metadata"}),
                additional_metadata=inject_traceparent_into_metadata(
                    dict(options.additional_metadata),
                ),
            )

            return wrapped(event_key, dict(payload), options)

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
        params = self.extract_bound_args(wrapped, args, kwargs)

        bulk_events = cast(list[BulkPushEventWithMetadata], params[0])
        options = cast(PushEventOptions, params[1])

        with self._tracer.start_as_current_span(
            "hatchet.bulk_push_event",
        ):
            options = PushEventOptions(
                **options.model_dump(exclude={"additional_metadata"}),
                additional_metadata=inject_traceparent_into_metadata(
                    dict(options.additional_metadata),
                ),
            )

            bulk_events_with_meta = [
                BulkPushEventWithMetadata(
                    **event.model_dump(exclude={"additional_metadata"}),
                    additional_metadata=inject_traceparent_into_metadata(
                        dict(event.additional_metadata),
                    ),
                )
                for event in bulk_events
            ]

            return wrapped(
                bulk_events_with_meta,
                options,
            )

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    def _wrap_run_workflow(
        self,
        wrapped: Callable[
            [str, JSONSerializableMapping, TriggerWorkflowOptions | None],
            WorkflowRunRef,
        ],
        instance: AdminClient,
        args: tuple[str, JSONSerializableMapping, TriggerWorkflowOptions | None],
        kwargs: dict[
            str, str | JSONSerializableMapping | TriggerWorkflowOptions | None
        ],
    ) -> WorkflowRunRef:
        params = self.extract_bound_args(wrapped, args, kwargs)

        workflow_name = cast(str, params[0])
        payload = cast(JSONSerializableMapping, params[1])
        options = cast(
            TriggerWorkflowOptions,
            params[2] if len(params) > 2 else TriggerWorkflowOptions(),
        )

        attributes = {
            OTelAttribute.RUN_WORKFLOW_WORKFLOW_NAME: workflow_name,
            OTelAttribute.RUN_WORKFLOW_PAYLOAD: json.dumps(payload, default=str),
            OTelAttribute.RUN_WORKFLOW_PARENT_ID: options.parent_id,
            OTelAttribute.RUN_WORKFLOW_PARENT_STEP_RUN_ID: options.parent_step_run_id,
            OTelAttribute.RUN_WORKFLOW_CHILD_INDEX: options.child_index,
            OTelAttribute.RUN_WORKFLOW_CHILD_KEY: options.child_key,
            OTelAttribute.RUN_WORKFLOW_NAMESPACE: options.namespace,
            OTelAttribute.RUN_WORKFLOW_ADDITIONAL_METADATA: json.dumps(
                options.additional_metadata, default=str
            ),
            OTelAttribute.RUN_WORKFLOW_PRIORITY: options.priority,
            OTelAttribute.RUN_WORKFLOW_DESIRED_WORKER_ID: options.desired_worker_id,
            OTelAttribute.RUN_WORKFLOW_STICKY: options.sticky,
            OTelAttribute.RUN_WORKFLOW_KEY: options.key,
        }

        with self._tracer.start_as_current_span(
            "hatchet.run_workflow",
            attributes={
                f"hatchet.{k.value}": v
                for k, v in attributes.items()
                if v and k not in self.config.otel.excluded_attributes
            },
        ):
            options = TriggerWorkflowOptions(
                **options.model_dump(exclude={"additional_metadata"}),
                additional_metadata=inject_traceparent_into_metadata(
                    dict(options.additional_metadata),
                ),
            )

            return wrapped(workflow_name, payload, options)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_async_run_workflow(
        self,
        wrapped: Callable[
            [str, JSONSerializableMapping, TriggerWorkflowOptions | None],
            Coroutine[None, None, WorkflowRunRef],
        ],
        instance: AdminClient,
        args: tuple[str, JSONSerializableMapping, TriggerWorkflowOptions | None],
        kwargs: dict[
            str, str | JSONSerializableMapping | TriggerWorkflowOptions | None
        ],
    ) -> WorkflowRunRef:
        params = self.extract_bound_args(wrapped, args, kwargs)

        workflow_name = cast(str, params[0])
        payload = cast(JSONSerializableMapping, params[1])
        options = cast(
            TriggerWorkflowOptions,
            params[2] if len(params) > 2 else TriggerWorkflowOptions(),
        )

        attributes = {
            OTelAttribute.RUN_WORKFLOW_WORKFLOW_NAME: workflow_name,
            OTelAttribute.RUN_WORKFLOW_PAYLOAD: json.dumps(payload, default=str),
            OTelAttribute.RUN_WORKFLOW_PARENT_ID: options.parent_id,
            OTelAttribute.RUN_WORKFLOW_PARENT_STEP_RUN_ID: options.parent_step_run_id,
            OTelAttribute.RUN_WORKFLOW_CHILD_INDEX: options.child_index,
            OTelAttribute.RUN_WORKFLOW_CHILD_KEY: options.child_key,
            OTelAttribute.RUN_WORKFLOW_NAMESPACE: options.namespace,
            OTelAttribute.RUN_WORKFLOW_ADDITIONAL_METADATA: json.dumps(
                options.additional_metadata, default=str
            ),
            OTelAttribute.RUN_WORKFLOW_PRIORITY: options.priority,
            OTelAttribute.RUN_WORKFLOW_DESIRED_WORKER_ID: options.desired_worker_id,
            OTelAttribute.RUN_WORKFLOW_STICKY: options.sticky,
            OTelAttribute.RUN_WORKFLOW_KEY: options.key,
        }

        with self._tracer.start_as_current_span(
            "hatchet.run_workflow",
            attributes={
                f"hatchet.{k.value}": v
                for k, v in attributes.items()
                if v and k not in self.config.otel.excluded_attributes
            },
        ):
            options = TriggerWorkflowOptions(
                **options.model_dump(exclude={"additional_metadata"}),
                additional_metadata=inject_traceparent_into_metadata(
                    dict(options.additional_metadata),
                ),
            )

            return await wrapped(workflow_name, payload, options)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    def _wrap_run_workflows(
        self,
        wrapped: Callable[
            [list[WorkflowRunTriggerConfig]],
            list[WorkflowRunRef],
        ],
        instance: AdminClient,
        args: tuple[list[WorkflowRunTriggerConfig],],
        kwargs: dict[str, list[WorkflowRunTriggerConfig]],
    ) -> list[WorkflowRunRef]:
        params = self.extract_bound_args(wrapped, args, kwargs)
        workflow_run_configs = cast(list[WorkflowRunTriggerConfig], params[0])

        with self._tracer.start_as_current_span(
            "hatchet.run_workflows",
        ):
            workflow_run_configs_with_meta = [
                WorkflowRunTriggerConfig(
                    **config.model_dump(exclude={"options"}),
                    options=TriggerWorkflowOptions(
                        **config.options.model_dump(exclude={"additional_metadata"}),
                        additional_metadata=inject_traceparent_into_metadata(
                            dict(config.options.additional_metadata),
                        ),
                    ),
                )
                for config in workflow_run_configs
            ]

            return wrapped(workflow_run_configs_with_meta)

    ## IMPORTANT: Keep these types in sync with the wrapped method's signature
    async def _wrap_async_run_workflows(
        self,
        wrapped: Callable[
            [list[WorkflowRunTriggerConfig]],
            Coroutine[None, None, list[WorkflowRunRef]],
        ],
        instance: AdminClient,
        args: tuple[list[WorkflowRunTriggerConfig],],
        kwargs: dict[str, list[WorkflowRunTriggerConfig]],
    ) -> list[WorkflowRunRef]:
        params = self.extract_bound_args(wrapped, args, kwargs)
        workflow_run_configs = cast(list[WorkflowRunTriggerConfig], params[0])

        with self._tracer.start_as_current_span(
            "hatchet.run_workflows",
        ):
            workflow_run_configs_with_meta = [
                WorkflowRunTriggerConfig(
                    **config.model_dump(exclude={"options"}),
                    options=TriggerWorkflowOptions(
                        **config.options.model_dump(exclude={"additional_metadata"}),
                        additional_metadata=inject_traceparent_into_metadata(
                            dict(config.options.additional_metadata),
                        ),
                    ),
                )
                for config in workflow_run_configs
            ]

            return await wrapped(workflow_run_configs_with_meta)

    def _uninstrument(self, **kwargs: InstrumentKwargs) -> None:
        self.tracer_provider = NoOpTracerProvider()
        self.meter_provider = NoOpMeterProvider()

        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_start_step_run")
        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_start_group_key_run")
        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_cancel_action")
        unwrap(hatchet_sdk, "clients.events.EventClient.push")
        unwrap(hatchet_sdk, "clients.events.EventClient.bulk_push")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.run_workflow")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.aio_run_workflow")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.run_workflows")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.aio_run_workflows")
