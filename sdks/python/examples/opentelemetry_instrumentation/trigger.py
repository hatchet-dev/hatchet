from opentelemetry.trace import get_tracer

from examples.opentelemetry_instrumentation.worker import otel_workflow
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HatchetInstrumentor().instrument()

tracer = get_tracer(__name__)


def main() -> None:
    # The run_workflow call is auto-traced with a "hatchet.run_workflow" span.
    # The traceparent is automatically injected into additional_metadata,
    # so the worker-side spans become children of this trigger span.
    with tracer.start_as_current_span("trigger_otel_data_pipeline"):
        result = otel_workflow.run(
            additional_metadata={
                "source": "otel-example",
                "pipeline": "data-ingest",
            },
        )
        print(f"Workflow result: {result}")


if __name__ == "__main__":
    main()
