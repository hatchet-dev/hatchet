import json
from collections.abc import Callable, Collection, Coroutine, Sequence
from datetime import datetime, timedelta, timezone
from importlib.metadata import version
from typing import Any, cast

import grpc

from hatchet_sdk.connection import load_channel_credentials
from hatchet_sdk.contracts import workflows_pb2 as v0_workflow_protos
from hatchet_sdk.utils.typing import JSONSerializableMapping

try:
    from opentelemetry.context import Context
    from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
    from opentelemetry.instrumentation.instrumentor import (  # type: ignore[attr-defined]
        BaseInstrumentor,
    )
    from opentelemetry.instrumentation.utils import unwrap
    from opentelemetry.metrics import MeterProvider, NoOpMeterProvider, get_meter
    from opentelemetry.sdk.trace import ReadableSpan, Span
    from opentelemetry.sdk.trace import TracerProvider as SDKTracerProvider
    from opentelemetry.sdk.trace.export import (
        BatchSpanProcessor,
        SpanExporter,
        SpanExportResult,
    )
    from opentelemetry.trace import (
        NoOpTracerProvider,
        SpanKind,
        StatusCode,
        TracerProvider,
        get_tracer,
        get_tracer_provider,
        set_tracer_provider,
    )
    from opentelemetry.trace.propagation.tracecontext import (
        TraceContextTextMapPropagator,
    )
    from wrapt import wrap_function_wrapper  # type: ignore[import-untyped]
except (RuntimeError, ImportError, ModuleNotFoundError) as e:
    raise ModuleNotFoundError(
        "To use the HatchetInstrumentor, you must install Hatchet's `otel` extra using (e.g.) `pip install hatchet-sdk[otel]`"
    ) from e

import inspect

import hatchet_sdk
from hatchet_sdk import ClientConfig
from hatchet_sdk.clients.admin import (
    AdminClient,
    ScheduleTriggerWorkflowOptions,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.clients.events import (
    BulkPushEventOptions,
    BulkPushEventWithMetadata,
    Event,
    EventClient,
    PushEventOptions,
    _inject_source_info,
)
from hatchet_sdk.context.context import DurableContext, DurableSpawnResult
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import Action
from hatchet_sdk.runnables.contextvars import ctx_hatchet_span_attributes
from hatchet_sdk.types.priority import Priority
from hatchet_sdk.types.trigger import TriggerWorkflowOptions
from hatchet_sdk.utils.opentelemetry import OTelAttribute
from hatchet_sdk.worker.runner.runner import Runner
from hatchet_sdk.workflow_run import WorkflowRunRef

hatchet_sdk_version = version("hatchet-sdk")


_RETRY_AFTER = timedelta(minutes=5)


class _HatchetSpanExporter(SpanExporter):
    """Wraps an OTLP exporter and silently backs off if the engine
    does not have the OTel collector service enabled (gRPC UNIMPLEMENTED).
    Retries periodically so that enabling o11y on the engine does not
    require a worker restart."""

    def __init__(self, inner: SpanExporter) -> None:
        self._inner = inner
        self._retry_at: datetime | None = None

    def export(self, spans: Sequence[ReadableSpan]) -> SpanExportResult:
        if self._retry_at and datetime.now(timezone.utc) < self._retry_at:
            return SpanExportResult.SUCCESS

        try:
            result = self._inner.export(spans)
            self._retry_at = None
            return result
        except Exception as exc:
            if _is_grpc_unimplemented(exc):
                self._retry_at = datetime.now(timezone.utc) + _RETRY_AFTER
                return SpanExportResult.SUCCESS
            raise

    def shutdown(self) -> None:
        self._inner.shutdown()

    def force_flush(self, timeout_millis: int = 30000) -> bool:
        return self._inner.force_flush(timeout_millis)


def _is_grpc_unimplemented(exc: Exception) -> bool:
    try:
        if isinstance(exc, grpc.RpcError):
            return exc.code() == grpc.StatusCode.UNIMPLEMENTED
    except ImportError:
        pass

    return "UNIMPLEMENTED" in str(exc)


class _HatchetAttributeSpanProcessor(BatchSpanProcessor):
    """SpanProcessor that injects hatchet.* attributes into every span
    created within a step run context, so that child spans are queryable
    by the same attributes (e.g. hatchet.step_run_id) as the parent."""

    def __init__(self, span_exporter: SpanExporter, **kwargs: Any) -> None:
        super().__init__(span_exporter, **kwargs)

    def on_start(self, span: Span, parent_context: Context | None = None) -> None:
        attrs = ctx_hatchet_span_attributes.get()
        if attrs and span.is_recording():
            existing = span.attributes or {}
            for key, value in attrs.items():
                if key not in existing:
                    span.set_attribute(key, value)
        super().on_start(span, parent_context)


InstrumentKwargs = TracerProvider | MeterProvider | None

OTEL_TRACEPARENT_KEY = "traceparent"


def create_traceparent() -> str | None:
    logger.warning(
        "as of SDK version 1.11.0, you no longer need to call `create_traceparent` manually. The traceparent will be automatically created by the instrumentor and injected into the metadata of actions and events when appropriate. This method will be removed in a future version.",
    )
    return _create_traceparent()


def _create_traceparent() -> str | None:
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
    logger.warning(
        "as of SDK version 1.11.0, you no longer need to call `parse_carrier_from_metadata` manually. This method will be removed in a future version.",
    )

    return _parse_carrier_from_metadata(metadata)


def _parse_carrier_from_metadata(
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
    >>> context = _parse_carrier_from_metadata(metadata)
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
    logger.warning(
        "as of SDK version 1.11.0, you no longer need to call `inject_traceparent_into_metadata` manually. The traceparent will automatically be injected by the instrumentor. This method will be removed in a future version.",
    )

    return _inject_traceparent_into_metadata(metadata, traceparent)


def _inject_traceparent_into_metadata(
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
        traceparent = _create_traceparent()

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

    :param tracer_provider: The OpenTelemetry TracerProvider to use.
            If not provided and `enable_hatchet_otel_collector` is True, a new SDKTracerProvider
            will be created. Otherwise, the global tracer provider will be used.
    :param meter_provider: The OpenTelemetry MeterProvider to use.
            If not provided, a no-op meter provider will be used.
    :param config: The configuration for the Hatchet client. If not provided,
            a default configuration will be used.
    :param enable_hatchet_otel_collector: If True (the default), adds an OTLP exporter
            to send traces to the Hatchet engine. Uses the same connection settings (host, TLS,
            token) as the Hatchet client. This can be combined with your own tracer_provider to
            send traces to multiple destinations (e.g., both Hatchet and Jaeger/Datadog).
    :param schedule_delay_millis: The delay in milliseconds between two consecutive
            exports of the BatchSpanProcessor. Defaults to the OTel SDK default (5000ms) if not set.
    :param max_export_batch_size: The maximum batch size for the BatchSpanProcessor.
            Defaults to the OTel SDK default (512) if not set.
    :param max_queue_size: The maximum queue size for the BatchSpanProcessor.
            Defaults to the OTel SDK default (2048) if not set.

    Example usage::

        # Send traces to Hatchet (default)
        instrumentor = HatchetInstrumentor()

        # Send traces to both Hatchet and your own collector
        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry.sdk.trace.export import BatchSpanProcessor
        from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

        provider = TracerProvider()
        provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint="your-collector:4317")))

        instrumentor = HatchetInstrumentor(
            tracer_provider=provider,  # Also sends to your collector
        )
    """

    def __init__(
        self,
        tracer_provider: TracerProvider | None = None,
        meter_provider: MeterProvider | None = None,
        config: ClientConfig | None = None,
        enable_hatchet_otel_collector: bool = True,
        schedule_delay_millis: int | None = None,
        max_export_batch_size: int | None = None,
        max_queue_size: int | None = None,
    ):
        self.config = config or ClientConfig()

        self._bsp_kwargs: dict[str, Any] = {}
        if schedule_delay_millis is not None:
            self._bsp_kwargs["schedule_delay_millis"] = schedule_delay_millis
        if max_export_batch_size is not None:
            self._bsp_kwargs["max_export_batch_size"] = max_export_batch_size
        if max_queue_size is not None:
            self._bsp_kwargs["max_queue_size"] = max_queue_size

        if tracer_provider is not None:
            self.tracer_provider = tracer_provider
        else:
            existing = get_tracer_provider()
            if isinstance(existing, SDKTracerProvider):
                self.tracer_provider = existing
            elif enable_hatchet_otel_collector:
                self.tracer_provider = SDKTracerProvider()
                set_tracer_provider(self.tracer_provider)
            else:
                self.tracer_provider = existing

        if enable_hatchet_otel_collector:
            self._add_hatchet_exporter()

        self.meter_provider = meter_provider or NoOpMeterProvider()

        super().__init__()

    def _add_hatchet_exporter(self) -> None:
        if not isinstance(self.tracer_provider, SDKTracerProvider):
            logger.warning(
                "enable_hatchet_otel_collector requires an opentelemetry.sdk.trace.TracerProvider. "
                "The provided tracer_provider does not support adding span processors. "
                "Traces will not be forwarded to Hatchet."
            )
            return

        endpoint = self.config.host_port
        insecure = self.config.tls_config.strategy == "none"
        headers = (("authorization", f"Bearer {self.config.token}"),)
        credentials = load_channel_credentials(self.config)

        otlp_exporter = OTLPSpanExporter(
            endpoint=endpoint,
            headers=headers,
            insecure=insecure,
            credentials=credentials,
        )

        self.tracer_provider.add_span_processor(
            _HatchetAttributeSpanProcessor(
                _HatchetSpanExporter(otlp_exporter), **self._bsp_kwargs
            )
        )

    def instrumentation_dependencies(self) -> Collection[str]:
        return ()

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

        ## IMPORTANT: We don't need to instrument the async version of `schedule_workflow`
        ## because it just calls the sync version internally.
        wrap_function_wrapper(
            hatchet_sdk,
            "clients.admin.AdminClient.schedule_workflow",
            self._wrap_schedule_workflow,
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

        wrap_function_wrapper(
            hatchet_sdk,
            "context.context.DurableContext.aio_wait_for",
            self._wrap_aio_wait_for,
        )

        wrap_function_wrapper(
            hatchet_sdk,
            "context.context.DurableContext._spawn_children_no_wait",
            self._wrap_spawn_children_no_wait,
        )

    def extract_bound_args(
        self,
        wrapped_func: Callable[..., Any],
        args: tuple[Any, ...],
        kwargs: dict[str, Any],
    ) -> list[Any]:
        try:
            import annotationlib  # type: ignore[import-not-found, unused-ignore]

            # Python 3.14+ with PEP 749 can fail evaluating annotations lazily,
            # so use Format.STRING to avoid resolving type hints.
            sig = inspect.signature(
                wrapped_func,
                annotation_format=annotationlib.Format.STRING,  # type: ignore[call-arg, unused-ignore]
            )
        except Exception:
            # Fallback for Python < 3.14 where annotation_format is not supported
            sig = inspect.signature(wrapped_func)

        bound_args = sig.bind(*args, **kwargs)
        bound_args.apply_defaults()

        return list(bound_args.arguments.values())

    async def _wrap_handle_start_step_run(
        self,
        wrapped: Callable[[Action], Coroutine[None, None, Exception | None]],
        instance: Runner,
        args: tuple[Action],
        kwargs: Any,
    ) -> Exception | None:
        params = self.extract_bound_args(wrapped, args, kwargs)

        action = cast(Action, params[0])

        traceparent = _parse_carrier_from_metadata(action.additional_metadata)
        span_name = "hatchet.start_step_run"

        if self.config.otel.include_task_name_in_start_step_run_span_name:
            span_name += f".{action.action_id}"

        hatchet_attrs = action.get_otel_attributes(self.config)
        token = ctx_hatchet_span_attributes.set(hatchet_attrs)

        try:
            with self._tracer.start_as_current_span(
                span_name,
                attributes=hatchet_attrs,
                context=traceparent,
                kind=SpanKind.CONSUMER,
            ) as span:
                result = await wrapped(*args, **kwargs)

                if isinstance(result, Exception):
                    span.record_exception(result)
                    span.set_status(StatusCode.ERROR, str(result))

                return result
        finally:
            ctx_hatchet_span_attributes.reset(token)

    async def _wrap_handle_cancel_action(
        self,
        wrapped: Callable[[Action], Coroutine[None, None, None]],
        instance: Runner,
        args: tuple[Action],
        kwargs: Any,
    ) -> None:
        action = args[0]

        with self._tracer.start_as_current_span(
            "hatchet.cancel_step_run",
            attributes={
                "instrumentor": "hatchet",
                "hatchet.step_run_id": action.step_run_id,
            },
            kind=SpanKind.CONSUMER,
        ):
            return await wrapped(*args, **kwargs)

    def _wrap_push_event(
        self,
        wrapped: Callable[..., Event],
        instance: EventClient,
        args: tuple[
            str,
            JSONSerializableMapping,
            PushEventOptions | None,
            JSONSerializableMapping | None,
            Priority | None,
            str | None,
        ],
        kwargs: dict[
            str,
            str | JSONSerializableMapping | PushEventOptions | Priority | None,
        ],
    ) -> Event:
        params = self.extract_bound_args(wrapped, args, kwargs)

        event_key = cast(str, params[0])
        payload = cast(JSONSerializableMapping, params[1])
        options = cast(PushEventOptions | None, params[2])
        additional_metadata = cast(JSONSerializableMapping | None, params[3])
        priority = cast(Priority | None, params[4])
        scope = cast(str | None, params[5])

        additional_metadata = additional_metadata or (
            options.additional_metadata if options else {}
        )

        priority_option = options.priority if options else None

        if isinstance(priority_option, int):
            priority_option = Priority(priority_option)

        priority = priority or priority_option
        scope = scope or (options.scope if options else None)

        attributes = {
            OTelAttribute.EVENT_KEY: event_key,
            OTelAttribute.ACTION_PAYLOAD: json.dumps(payload, default=str),
            OTelAttribute.ADDITIONAL_METADATA: json.dumps(
                additional_metadata, default=str
            ),
            OTelAttribute.PRIORITY: priority,
            OTelAttribute.FILTER_SCOPE: scope,
        }

        with self._tracer.start_as_current_span(
            "hatchet.push_event",
            attributes={
                "instrumentor": "hatchet",
                **{
                    f"hatchet.{k.value}": v
                    for k, v in attributes.items()
                    if v
                    and k not in self.config.otel.excluded_attributes
                    and v != "{}"
                    and v != "[]"
                },
            },
            kind=SpanKind.PRODUCER,
        ):
            return wrapped(
                event_key,
                payload,
                None,
                _inject_source_info(
                    _inject_traceparent_into_metadata(dict(additional_metadata)),
                ),
                priority,
                scope,
            )

    def _wrap_bulk_push_event(
        self,
        wrapped: Callable[
            [list[BulkPushEventWithMetadata], BulkPushEventOptions | None], list[Event]
        ],
        instance: EventClient,
        args: tuple[
            list[BulkPushEventWithMetadata],
            BulkPushEventOptions | None,
        ],
        kwargs: dict[
            str, list[BulkPushEventWithMetadata] | BulkPushEventOptions | None
        ],
    ) -> list[Event]:
        params = self.extract_bound_args(wrapped, args, kwargs)

        bulk_events = cast(list[BulkPushEventWithMetadata], params[0])
        options = cast(BulkPushEventOptions | None, params[1])

        num_bulk_events = len(bulk_events)
        unique_event_keys = {event.key for event in bulk_events}

        with self._tracer.start_as_current_span(
            "hatchet.bulk_push_event",
            attributes={
                "instrumentor": "hatchet",
                "hatchet.num_events": num_bulk_events,
                "hatchet.unique_event_keys": json.dumps(unique_event_keys, default=str),
            },
            kind=SpanKind.PRODUCER,
        ):
            bulk_events_with_meta = [
                BulkPushEventWithMetadata(
                    **event.model_dump(exclude={"additional_metadata"}),
                    additional_metadata=_inject_source_info(
                        _inject_traceparent_into_metadata(
                            event.additional_metadata,
                        )
                    ),
                )
                for event in bulk_events
            ]

            return wrapped(
                bulk_events_with_meta,
                options,
            )

    def _wrap_run_workflow(
        self,
        wrapped: Callable[
            [str, str | None, TriggerWorkflowOptions],
            WorkflowRunRef,
        ],
        instance: AdminClient,
        args: tuple[str, str | None, TriggerWorkflowOptions],
        kwargs: dict[str, str | None | TriggerWorkflowOptions],
    ) -> WorkflowRunRef:
        params = self.extract_bound_args(wrapped, args, kwargs)

        workflow_name = cast(str, params[0])
        payload = cast(str | None, params[1])
        options = cast(
            TriggerWorkflowOptions,
            params[2] if len(params) > 2 else TriggerWorkflowOptions(),
        )

        attributes = {
            OTelAttribute.WORKFLOW_NAME: workflow_name,
            OTelAttribute.ACTION_PAYLOAD: payload,
            OTelAttribute.PARENT_ID: options.parent_id,
            OTelAttribute.PARENT_STEP_RUN_ID: options.parent_step_run_id,
            OTelAttribute.CHILD_INDEX: options.child_index,
            OTelAttribute.CHILD_KEY: options.child_key,
            OTelAttribute.NAMESPACE: options.namespace,
            OTelAttribute.ADDITIONAL_METADATA: json.dumps(
                options.additional_metadata, default=str
            ),
            OTelAttribute.PRIORITY: options.priority,
            OTelAttribute.DESIRED_WORKER_ID: options.desired_worker_id,
            OTelAttribute.STICKY: options.sticky,
            OTelAttribute.KEY: options.key,
        }

        with self._tracer.start_as_current_span(
            "hatchet.run_workflow",
            attributes={
                "instrumentor": "hatchet",
                **{
                    f"hatchet.{k.value}": v
                    for k, v in attributes.items()
                    if v
                    and k not in self.config.otel.excluded_attributes
                    and v != "{}"
                    and v != "[]"
                },
            },
            kind=SpanKind.PRODUCER,
        ):
            options = TriggerWorkflowOptions(
                **options.model_dump(exclude={"additional_metadata"}),
                additional_metadata=_inject_traceparent_into_metadata(
                    options.additional_metadata,
                ),
            )

            return wrapped(workflow_name, payload, options)

    async def _wrap_async_run_workflow(
        self,
        wrapped: Callable[
            [str, str | None, TriggerWorkflowOptions],
            Coroutine[None, None, WorkflowRunRef],
        ],
        instance: AdminClient,
        args: tuple[str, str | None, TriggerWorkflowOptions],
        kwargs: dict[str, str | None | TriggerWorkflowOptions],
    ) -> WorkflowRunRef:
        params = self.extract_bound_args(wrapped, args, kwargs)

        workflow_name = cast(str, params[0])
        payload = cast(str | None, params[1])
        options = cast(
            TriggerWorkflowOptions,
            params[2] if len(params) > 2 else TriggerWorkflowOptions(),
        )

        attributes = {
            OTelAttribute.WORKFLOW_NAME: workflow_name,
            OTelAttribute.ACTION_PAYLOAD: payload,
            OTelAttribute.PARENT_ID: options.parent_id,
            OTelAttribute.PARENT_STEP_RUN_ID: options.parent_step_run_id,
            OTelAttribute.CHILD_INDEX: options.child_index,
            OTelAttribute.CHILD_KEY: options.child_key,
            OTelAttribute.NAMESPACE: options.namespace,
            OTelAttribute.ADDITIONAL_METADATA: json.dumps(
                options.additional_metadata, default=str
            ),
            OTelAttribute.PRIORITY: options.priority,
            OTelAttribute.DESIRED_WORKER_ID: options.desired_worker_id,
            OTelAttribute.STICKY: options.sticky,
            OTelAttribute.KEY: options.key,
        }

        with self._tracer.start_as_current_span(
            "hatchet.run_workflow",
            attributes={
                "instrumentor": "hatchet",
                **{
                    f"hatchet.{k.value}": v
                    for k, v in attributes.items()
                    if v
                    and k not in self.config.otel.excluded_attributes
                    and v != "{}"
                    and v != "[]"
                },
            },
            kind=SpanKind.PRODUCER,
        ):
            options = TriggerWorkflowOptions(
                **options.model_dump(exclude={"additional_metadata"}),
                additional_metadata=_inject_traceparent_into_metadata(
                    options.additional_metadata,
                ),
            )

            return await wrapped(workflow_name, payload, options)

    def _wrap_schedule_workflow(
        self,
        wrapped: Callable[
            [
                str,
                list[datetime],
                str | None,
                ScheduleTriggerWorkflowOptions | None,
            ],
            v0_workflow_protos.WorkflowVersion,
        ],
        instance: AdminClient,
        args: tuple[
            str,
            list[datetime],
            str | None,
            ScheduleTriggerWorkflowOptions | None,
        ],
        kwargs: dict[
            str,
            str | None | list[datetime] | ScheduleTriggerWorkflowOptions,
        ],
    ) -> v0_workflow_protos.WorkflowVersion:
        params = self.extract_bound_args(wrapped, args, kwargs)

        workflow_name = cast(str, params[0])
        schedules = cast(list[datetime], params[1])
        input = cast(str | None, params[2])
        options = cast(
            ScheduleTriggerWorkflowOptions,
            params[3] if len(params) > 3 else ScheduleTriggerWorkflowOptions(),
        )

        attributes = {
            OTelAttribute.WORKFLOW_NAME: workflow_name,
            OTelAttribute.RUN_AT_TIMESTAMPS: json.dumps(
                [ts.isoformat() for ts in schedules]
            ),
            OTelAttribute.ACTION_PAYLOAD: input,
            OTelAttribute.PARENT_ID: options.parent_id,
            OTelAttribute.PARENT_STEP_RUN_ID: options.parent_step_run_id,
            OTelAttribute.CHILD_INDEX: options.child_index,
            OTelAttribute.CHILD_KEY: options.child_key,
            OTelAttribute.NAMESPACE: options.namespace,
            OTelAttribute.ADDITIONAL_METADATA: json.dumps(
                options.additional_metadata, default=str
            ),
            OTelAttribute.PRIORITY: options.priority,
        }

        with self._tracer.start_as_current_span(
            "hatchet.schedule_workflow",
            attributes={
                "instrumentor": "hatchet",
                **{
                    f"hatchet.{k.value}": v
                    for k, v in attributes.items()
                    if v
                    and k not in self.config.otel.excluded_attributes
                    and v != "{}"
                    and v != "[]"
                },
            },
            kind=SpanKind.PRODUCER,
        ):
            options = ScheduleTriggerWorkflowOptions(
                **options.model_dump(exclude={"additional_metadata"}),
                additional_metadata=_inject_traceparent_into_metadata(
                    options.additional_metadata,
                ),
            )

            return wrapped(workflow_name, schedules, input, options)

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

        num_workflows = len(workflow_run_configs)
        unique_workflow_names = {
            config.workflow_name for config in workflow_run_configs
        }

        with self._tracer.start_as_current_span(
            "hatchet.run_workflows",
            attributes={
                "instrumentor": "hatchet",
                "hatchet.num_workflows": num_workflows,
                "hatchet.unique_workflow_names": json.dumps(
                    unique_workflow_names, default=str
                ),
            },
            kind=SpanKind.PRODUCER,
        ):
            workflow_run_configs_with_meta = [
                WorkflowRunTriggerConfig(
                    **config.model_dump(exclude={"options"}),
                    options=TriggerWorkflowOptions(
                        **config.options.model_dump(exclude={"additional_metadata"}),
                        additional_metadata=_inject_traceparent_into_metadata(
                            config.options.additional_metadata,
                        ),
                    ),
                )
                for config in workflow_run_configs
            ]

            return wrapped(workflow_run_configs_with_meta)

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
        num_workflows = len(workflow_run_configs)
        unique_workflow_names = {
            config.workflow_name for config in workflow_run_configs
        }

        with self._tracer.start_as_current_span(
            "hatchet.run_workflows",
            attributes={
                "instrumentor": "hatchet",
                "hatchet.num_workflows": num_workflows,
                "hatchet.unique_workflow_names": json.dumps(
                    unique_workflow_names, default=str
                ),
            },
            kind=SpanKind.PRODUCER,
        ):
            workflow_run_configs_with_meta = [
                WorkflowRunTriggerConfig(
                    **config.model_dump(exclude={"options"}),
                    options=TriggerWorkflowOptions(
                        **config.options.model_dump(exclude={"additional_metadata"}),
                        additional_metadata=_inject_traceparent_into_metadata(
                            config.options.additional_metadata,
                        ),
                    ),
                )
                for config in workflow_run_configs
            ]

            return await wrapped(workflow_run_configs_with_meta)

    async def _wrap_aio_wait_for(
        self,
        wrapped: Callable[..., Coroutine[None, None, dict[str, Any]]],
        instance: DurableContext,
        args: tuple[Any, ...],
        kwargs: dict[str, Any],
    ) -> dict[str, Any]:
        params = self.extract_bound_args(wrapped, args, kwargs)

        signal_key = cast(str, params[0])
        conditions = params[1:]

        traceparent = _parse_carrier_from_metadata(instance.action.additional_metadata)

        attributes: dict[OTelAttribute, str | int | None] = {
            OTelAttribute.SIGNAL_KEY: signal_key,
            OTelAttribute.NUM_CONDITIONS: len(conditions),
            OTelAttribute.STEP_RUN_ID: instance.step_run_id,
        }

        with self._tracer.start_as_current_span(
            "hatchet.durable.wait_for",
            attributes={
                "instrumentor": "hatchet",
                **{
                    f"hatchet.{k.value}": v
                    for k, v in attributes.items()
                    if v is not None and k not in self.config.otel.excluded_attributes
                },
            },
            context=traceparent,
            kind=SpanKind.INTERNAL,
        ) as span:
            try:
                return await wrapped(*args, **kwargs)
            except Exception as e:
                span.set_status(StatusCode.ERROR, str(e))
                raise

    async def _wrap_spawn_children_no_wait(
        self,
        wrapped: Callable[..., Coroutine[None, None, list[DurableSpawnResult]]],
        instance: DurableContext,
        args: tuple[Any, ...],
        kwargs: dict[str, Any],
    ) -> list[DurableSpawnResult]:
        params = self.extract_bound_args(wrapped, args, kwargs)

        configs = cast(list[WorkflowRunTriggerConfig], params[0])

        traceparent = _parse_carrier_from_metadata(instance.action.additional_metadata)

        if len(configs) == 1:
            config = configs[0]
            span_name = "hatchet.run_workflow"
            span_attributes: dict[str, str | int] = {
                "instrumentor": "hatchet",
                **{
                    f"hatchet.{k.value}": v
                    for k, v in {
                        OTelAttribute.WORKFLOW_NAME: config.workflow_name,
                        OTelAttribute.ACTION_PAYLOAD: config.input,
                        OTelAttribute.PARENT_ID: config.options.parent_id,
                        OTelAttribute.PARENT_STEP_RUN_ID: config.options.parent_step_run_id,
                        OTelAttribute.CHILD_INDEX: config.options.child_index,
                        OTelAttribute.CHILD_KEY: config.options.child_key,
                        OTelAttribute.NAMESPACE: config.options.namespace,
                        OTelAttribute.ADDITIONAL_METADATA: json.dumps(
                            config.options.additional_metadata, default=str
                        ),
                        OTelAttribute.PRIORITY: config.options.priority,
                        OTelAttribute.DESIRED_WORKER_ID: config.options.desired_worker_id,
                        OTelAttribute.STICKY: config.options.sticky,
                        OTelAttribute.KEY: config.options.key,
                    }.items()
                    if v
                    and k not in self.config.otel.excluded_attributes
                    and v != "{}"
                    and v != "[]"
                },
            }
        else:
            unique_workflow_names = {c.workflow_name for c in configs}
            span_name = "hatchet.run_workflows"
            span_attributes = {
                "instrumentor": "hatchet",
                "hatchet.num_workflows": len(configs),
                "hatchet.unique_workflow_names": json.dumps(
                    unique_workflow_names, default=str
                ),
            }

        with self._tracer.start_as_current_span(
            span_name,
            attributes=span_attributes,
            context=traceparent,
            kind=SpanKind.PRODUCER,
        ) as span:
            try:
                return await wrapped(*args, **kwargs)
            except Exception as e:
                span.set_status(StatusCode.ERROR, str(e))
                raise

    def _uninstrument(self, **kwargs: InstrumentKwargs) -> None:
        self.tracer_provider = NoOpTracerProvider()
        self.meter_provider = NoOpMeterProvider()

        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_start_step_run")
        unwrap(hatchet_sdk, "worker.runner.runner.Runner.handle_cancel_action")
        unwrap(hatchet_sdk, "clients.events.EventClient.push")
        unwrap(hatchet_sdk, "clients.events.EventClient.bulk_push")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.run_workflow")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.aio_run_workflow")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.schedule_workflow")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.run_workflows")
        unwrap(hatchet_sdk, "clients.admin.AdminClient.aio_run_workflows")
        unwrap(hatchet_sdk, "context.context.DurableContext.aio_wait_for")
        unwrap(hatchet_sdk, "context.context.DurableContext._spawn_children_no_wait")
