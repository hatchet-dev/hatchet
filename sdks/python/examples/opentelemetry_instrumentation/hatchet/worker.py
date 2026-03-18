"""
HatchetInstrumentor example with rich traces.

Run the worker:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.worker

Then trigger it from another terminal:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.trigger
"""

import time

from pydantic import BaseModel

from opentelemetry.trace import StatusCode, get_tracer

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

# > Setup
HatchetInstrumentor().instrument()

hatchet = Hatchet()
# !!


otel_workflow = hatchet.workflow(name="OTelDataPipeline")


class FetchDataOutput(BaseModel):
    records_fetched: int


class ValidateDataOutput(BaseModel):
    valid_records: int
    dropped: int


class ProcessDataOutput(BaseModel):
    processed_groups: int


class SaveResultsOutput(BaseModel):
    saved: bool


# > Custom Spans
@otel_workflow.task()
def fetch_data(input: EmptyModel, ctx: Context) -> FetchDataOutput:
    tracer = get_tracer(__name__)

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

    return FetchDataOutput(records_fetched=42)
# !!


@otel_workflow.task()
def validate_data(input: EmptyModel, ctx: Context) -> ValidateDataOutput:
    tracer = get_tracer(__name__)

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

    return ValidateDataOutput(valid_records=40, dropped=2)


@otel_workflow.task()
def process_data(input: EmptyModel, ctx: Context) -> ProcessDataOutput:
    tracer = get_tracer(__name__)

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

    return ProcessDataOutput(processed_groups=8)


@otel_workflow.task()
def save_results(input: EmptyModel, ctx: Context) -> SaveResultsOutput:
    tracer = get_tracer(__name__)

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

    return SaveResultsOutput(saved=True)


# > Worker
def main() -> None:
    worker = hatchet.worker(
        "otel-pipeline-worker",
        workflows=[otel_workflow],
    )
    worker.start()
# !!


if __name__ == "__main__":
    main()
