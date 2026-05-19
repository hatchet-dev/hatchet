"""Trigger the HyperDX example workflows and print results."""

import asyncio

from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import get_tracer, set_tracer_provider

from hatchet_sdk import Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HYPERDX_OTEL_ENDPOINT = "localhost:4317"

provider = TracerProvider()
provider.add_span_processor(
    BatchSpanProcessor(OTLPSpanExporter(endpoint=HYPERDX_OTEL_ENDPOINT, insecure=True))
)

set_tracer_provider(provider)
HatchetInstrumentor(tracer_provider=provider).instrument()

hatchet = Hatchet()


async def main() -> None:
    from examples.opentelemetry_instrumentation.hyperdx.worker import (
        TaskInput,
        hyperdx_parent_task,
        hyperdx_task,
    )

    tracer = get_tracer("hyperdx-trigger")

    # 1. Direct workflow run
    with tracer.start_as_current_span("trigger-direct"):
        result = await hyperdx_task.aio_run()
        print(f"Direct: {result}")

    # 2. Parent task that pushes an event and spawns a child
    with tracer.start_as_current_span("trigger-parent-child"):
        result = await hyperdx_parent_task.aio_run(
            input=TaskInput(message="hello from parent"),
        )
        print(f"Parent: {result}")

    provider.force_flush()


if __name__ == "__main__":
    asyncio.run(main())
