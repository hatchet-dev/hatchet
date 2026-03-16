"""
Trigger the OTelOrderProcessing DAG workflow.

Make sure the worker is already running:
    poetry run python -m examples.opentelemetry_dag.worker

Then run this:
    poetry run python -m examples.opentelemetry_dag.trigger
"""

from opentelemetry.trace import get_tracer

from examples.opentelemetry_dag.worker import order_workflow
from hatchet_sdk.clients.admin import TriggerWorkflowOptions
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HatchetInstrumentor(enable_hatchet_otel_collector=True).instrument()

tracer = get_tracer(__name__)


def main() -> None:
    with tracer.start_as_current_span("trigger_order_processing"):
        result = order_workflow.run(
            options=TriggerWorkflowOptions(
                additional_metadata={
                    "source": "otel-dag-example",
                    "order_id": "order-123",
                },
            ),
        )
        print(f"Workflow result: {result}")


if __name__ == "__main__":
    main()
