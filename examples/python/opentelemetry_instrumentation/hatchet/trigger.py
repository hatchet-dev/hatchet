"""
Trigger the OTelDataPipeline workflow.

Make sure the worker is already running:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.worker

Then run this:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.trigger
"""

from opentelemetry.trace import get_tracer

from examples.opentelemetry_instrumentation.hatchet.worker import otel_workflow
from hatchet_sdk.clients.admin import TriggerWorkflowOptions
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HatchetInstrumentor(enable_hatchet_otel_collector=True).instrument()

tracer = get_tracer(__name__)


def main() -> None:
    # The run_workflow call is auto-traced with a "hatchet.run_workflow" span.
    # The traceparent is automatically injected into additional_metadata,
    # so the worker-side spans become children of this trigger span.
    with tracer.start_as_current_span("trigger_otel_data_pipeline"):
        result = otel_workflow.run(
            options=TriggerWorkflowOptions(
                additional_metadata={
                    "source": "otel-example",
                    "pipeline": "data-ingest",
                },
            ),
        )
        print(f"Workflow result: {result}")


if __name__ == "__main__":
    main()
