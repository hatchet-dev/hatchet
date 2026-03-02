"""
Simple HatchetInstrumentor example.

Run the worker:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.worker

Then trigger it from another terminal:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.trigger
"""

from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import ConsoleSpanExporter, SimpleSpanProcessor

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

hatchet = Hatchet()

simple_otel_workflow = hatchet.workflow(name="SimpleOTelWorkflow")


@simple_otel_workflow.task()
def step_one(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("step_one: doing work")
    return {"step": "one"}


@simple_otel_workflow.task()
def step_two(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("step_two: doing work")
    return {"step": "two"}


def main() -> None:
    # Set up a TracerProvider that prints spans to the console so you can see them
    resource = Resource(attributes={SERVICE_NAME: "hatchet-otel-simple-example"})
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(SimpleSpanProcessor(ConsoleSpanExporter()))

    # Instrument Hatchet — must happen before starting the worker
    HatchetInstrumentor(
        tracer_provider=provider,
        enable_hatchet_otel_collector=True,
    ).instrument()

    worker = hatchet.worker(
        "otel-simple-worker",
        workflows=[simple_otel_workflow],
    )
    worker.start()


if __name__ == "__main__":
    main()
