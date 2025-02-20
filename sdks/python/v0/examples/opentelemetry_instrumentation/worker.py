from examples.opentelemetry_instrumentation.client import hatchet
from examples.opentelemetry_instrumentation.tracer import trace_provider
from hatchet_sdk import Context
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HatchetInstrumentor(
    tracer_provider=trace_provider,
).instrument()


@hatchet.workflow(on_events=["otel:event"])
class OTelWorkflow:
    @hatchet.step()
    def your_spans_are_children_of_hatchet_span(
        self, context: Context
    ) -> dict[str, str]:
        with trace_provider.get_tracer(__name__).start_as_current_span("step1"):
            print("executed step")
            return {
                "foo": "bar",
            }

    @hatchet.step()
    def your_spans_are_still_children_of_hatchet_span(self, context: Context) -> None:
        with trace_provider.get_tracer(__name__).start_as_current_span("step2"):
            raise Exception("Manually instrumented step failed failed")

    @hatchet.step()
    def this_step_is_still_instrumented(self, context: Context) -> dict[str, str]:
        print("executed still-instrumented step")
        return {
            "still": "instrumented",
        }

    @hatchet.step()
    def this_step_is_also_still_instrumented(self, context: Context) -> None:
        raise Exception("Still-instrumented step failed")


def main() -> None:
    worker = hatchet.worker("otel-example-worker", max_runs=1)
    worker.register_workflow(OTelWorkflow())
    worker.start()


if __name__ == "__main__":
    main()
