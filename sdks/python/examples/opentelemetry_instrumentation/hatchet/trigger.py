"""
Trigger the order processing workflow.

Make sure the worker is already running:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.worker

Then run this:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.trigger
"""

from opentelemetry.trace import get_tracer

from examples.opentelemetry_instrumentation.hatchet.worker import otel_workflow
from hatchet_sdk.clients.admin import TriggerWorkflowOptions
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HatchetInstrumentor().instrument()

tracer = get_tracer(__name__)


# > Trigger
def main() -> None:
    with tracer.start_as_current_span("trigger_order_processing"):
        result = otel_workflow.run(
            options=TriggerWorkflowOptions(
                additional_metadata={
                    "source": "otel-example",
                },
            ),
        )
        print(f"Workflow result: {result}")


# !!


if __name__ == "__main__":
    main()
