from examples.opentelemetry_instrumentation.client import hatchet
from examples.opentelemetry_instrumentation.tracer import trace_provider
from hatchet_sdk import Context, EmptyModel
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HatchetInstrumentor(
    tracer_provider=trace_provider,
).instrument()

otel_workflow = hatchet.workflow(
    name="OTelWorkflow",
)


@otel_workflow.task()
def your_spans_are_children_of_hatchet_span(
    input: EmptyModel, ctx: Context
) -> dict[str, str]:
    with trace_provider.get_tracer(__name__).start_as_current_span("step1"):
        print("executed step")
        return {
            "foo": "bar",
        }


@otel_workflow.task()
def your_spans_are_still_children_of_hatchet_span(
    input: EmptyModel, ctx: Context
) -> None:
    with trace_provider.get_tracer(__name__).start_as_current_span("step2"):
        raise Exception("Manually instrumented step failed failed")


@otel_workflow.task()
def this_step_is_still_instrumented(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("executed still-instrumented step")
    return {
        "still": "instrumented",
    }


@otel_workflow.task()
def this_step_is_also_still_instrumented(input: EmptyModel, ctx: Context) -> None:
    raise Exception("Still-instrumented step failed")


def main() -> None:
    worker = hatchet.worker("otel-example-worker", slots=1, workflows=[otel_workflow])
    worker.start()


if __name__ == "__main__":
    main()
