"""
Example: export Hatchet traces to a local HyperDX instance.

Prerequisites:
    cd sdks/python/examples/opentelemetry_instrumentation/hyperdx
    docker compose up -d

Then run this worker:
    uv run python -m examples.opentelemetry_instrumentation.hyperdx.worker

Open http://localhost:8088 to view traces in HyperDX.
"""

import time
from typing import Any

from pydantic import BaseModel

from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import get_tracer, set_tracer_provider

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HYPERDX_OTEL_ENDPOINT = "localhost:4317"

provider = TracerProvider()
provider.add_span_processor(
    BatchSpanProcessor(OTLPSpanExporter(endpoint=HYPERDX_OTEL_ENDPOINT, insecure=True))
)

set_tracer_provider(provider)
HatchetInstrumentor(tracer_provider=provider).instrument()

hatchet = Hatchet()


class TaskInput(BaseModel):
    message: str = "hello"


@hatchet.task()
def hyperdx_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    tracer = get_tracer("hyperdx-example")
    with tracer.start_as_current_span("do-work") as span:
        span.set_attribute("example.iteration", 1)
        time.sleep(0.05)
    return {"status": "ok"}


@hatchet.task(on_events=["hyperdx:test-event"], input_validator=TaskInput)
def hyperdx_event_task(input: TaskInput, ctx: Context) -> dict[str, str]:
    tracer = get_tracer("hyperdx-example")
    with tracer.start_as_current_span("handle-event") as span:
        span.set_attribute("input.message", input.message)
        time.sleep(0.03)
    return {"status": "event-handled"}


@hatchet.task(input_validator=TaskInput)
def hyperdx_child_task(input: TaskInput, ctx: Context) -> dict[str, str]:
    tracer = get_tracer("hyperdx-example")
    with tracer.start_as_current_span("child-work") as span:
        span.set_attribute("child.message", input.message)
        time.sleep(0.02)
    return {"status": "child-done"}


@hatchet.task(input_validator=TaskInput)
async def hyperdx_parent_task(input: TaskInput, ctx: Context) -> dict[str, Any]:
    tracer = get_tracer("hyperdx-example")

    with tracer.start_as_current_span("push-event") as span:
        span.set_attribute("event.key", "hyperdx:test-event")
        hatchet.event.push(
            "hyperdx:test-event",
            {"message": f"from parent: {input.message}"},
        )

    with tracer.start_as_current_span("spawn-child") as span:
        span.set_attribute("parent.message", input.message)
        result = await hyperdx_child_task.aio_run(
            input=TaskInput(message=f"from parent: {input.message}"),
        )

    return {"child_result": result}


def main() -> None:
    worker = hatchet.worker(
        "hyperdx-example-worker",
        workflows=[
            hyperdx_task,
            hyperdx_event_task,
            hyperdx_child_task,
            hyperdx_parent_task,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
