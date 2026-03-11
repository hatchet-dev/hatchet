"""
HatchetInstrumentor example with rich traces.

Run the worker:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.worker

Then trigger it from another terminal:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.trigger
"""

import time

from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import ConsoleSpanExporter, SimpleSpanProcessor
from opentelemetry.trace import StatusCode, Tracer

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

hatchet = Hatchet()

otel_workflow = hatchet.workflow(name="OTelDataPipeline")

# Module-level tracer — will be set in main() before the worker starts.
# Tasks use this to create custom child spans inside the auto-instrumented
# hatchet task run parent span.
_tracer: Tracer | None = None


def _get_tracer() -> Tracer:
    global _tracer
    if _tracer is None:
        from opentelemetry.trace import get_tracer

        _tracer = get_tracer(__name__)
    return _tracer


@otel_workflow.task()
def fetch_data(input: EmptyModel, ctx: Context) -> dict[str, str]:
    tracer = _get_tracer()

    with tracer.start_as_current_span(
        "http.request",
        attributes={"http.method": "GET", "http.url": "https://api.example.com/data"},
    ) as span:
        time.sleep(0.05)
        span.set_attribute("http.status_code", 200)
        span.set_attribute("http.response_content_length", 4096)

    with tracer.start_as_current_span("json.parse") as span:
        time.sleep(0.01)
        span.set_attribute("json.record_count", 42)

    return {"records_fetched": "42"}


@otel_workflow.task()
def validate_data(input: EmptyModel, ctx: Context) -> dict[str, str]:
    tracer = _get_tracer()

    with tracer.start_as_current_span("schema.validate") as span:
        time.sleep(0.02)
        span.set_attribute("validation.schema", "v2.1")
        span.set_attribute("validation.records_checked", 42)
        span.set_attribute("validation.errors", 2)
        span.set_status(StatusCode.OK, "2 records failed validation")

    with tracer.start_as_current_span("data.clean") as span:
        time.sleep(0.01)
        span.set_attribute("clean.records_dropped", 2)
        span.set_attribute("clean.records_remaining", 40)

    return {"valid_records": "40", "dropped": "2"}


@otel_workflow.task()
def process_data(input: EmptyModel, ctx: Context) -> dict[str, str]:
    tracer = _get_tracer()

    with tracer.start_as_current_span("transform.pipeline") as pipeline_span:
        pipeline_span.set_attribute("pipeline.stages", 3)

        with tracer.start_as_current_span("transform.normalize"):
            time.sleep(0.015)

        with tracer.start_as_current_span("transform.enrich") as enrich_span:
            time.sleep(0.02)
            enrich_span.set_attribute("enrich.source", "geocoding-api")

        with tracer.start_as_current_span("transform.aggregate") as agg_span:
            time.sleep(0.03)
            agg_span.set_attribute("aggregate.groups", 8)
            agg_span.set_attribute("aggregate.method", "sum")

    return {"processed_groups": "8"}


@otel_workflow.task()
def save_results(input: EmptyModel, ctx: Context) -> dict[str, str]:
    tracer = _get_tracer()

    with tracer.start_as_current_span(
        "db.query",
        attributes={"db.system": "postgresql", "db.operation": "INSERT"},
    ) as span:
        time.sleep(0.04)
        span.set_attribute("db.rows_affected", 8)

    with tracer.start_as_current_span("cache.invalidate") as span:
        time.sleep(0.005)
        span.set_attribute("cache.keys_invalidated", 3)

    with tracer.start_as_current_span("notification.send") as span:
        time.sleep(0.01)
        span.set_attribute("notification.channel", "webhook")
        span.set_attribute("notification.status", "delivered")

    return {"saved": "true"}


def main() -> None:
    resource = Resource(attributes={SERVICE_NAME: "hatchet-otel-pipeline-example"})
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(SimpleSpanProcessor(ConsoleSpanExporter()))

    HatchetInstrumentor(
        tracer_provider=provider,
        enable_hatchet_otel_collector=True,
    ).instrument()

    global _tracer
    _tracer = provider.get_tracer(__name__)

    worker = hatchet.worker(
        "otel-pipeline-worker",
        workflows=[otel_workflow],
    )
    worker.start()


if __name__ == "__main__":
    main()
