"""
Trigger the SimpleOTelWorkflow.

Make sure the worker is already running:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.worker

Then run this:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.trigger
"""

from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import ConsoleSpanExporter, SimpleSpanProcessor

from examples.opentelemetry_instrumentation.hatchet.worker import simple_otel_workflow
from hatchet_sdk.clients.admin import TriggerWorkflowOptions
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

# Use the same console exporter so you can see trigger-side spans too
resource = Resource(attributes={SERVICE_NAME: "hatchet-otel-simple-trigger"})
provider = TracerProvider(resource=resource)
provider.add_span_processor(SimpleSpanProcessor(ConsoleSpanExporter()))

HatchetInstrumentor(
    tracer_provider=provider,
    enable_hatchet_otel_collector=True,
).instrument()

tracer = provider.get_tracer(__name__)


def main() -> None:
    # The run_workflow call is auto-traced with a "hatchet.run_workflow" span.
    # The traceparent is automatically injected into additional_metadata,
    # so the worker-side spans become children of this trigger span.
    with tracer.start_as_current_span("trigger_simple_otel_workflow"):
        result = simple_otel_workflow.run(
            options=TriggerWorkflowOptions(
                additional_metadata={"source": "otel-example"},
            ),
        )
        print(f"Workflow result: {result}")


if __name__ == "__main__":
    main()
