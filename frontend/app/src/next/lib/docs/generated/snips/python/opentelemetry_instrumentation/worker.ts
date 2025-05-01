import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from examples.opentelemetry_instrumentation.client import hatchet\nfrom examples.opentelemetry_instrumentation.tracer import trace_provider\nfrom hatchet_sdk import Context, EmptyModel\nfrom hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor\n\nHatchetInstrumentor(\n    tracer_provider=trace_provider,\n).instrument()\n\notel_workflow = hatchet.workflow(\n    name='OTelWorkflow',\n)\n\n\n@otel_workflow.task()\ndef your_spans_are_children_of_hatchet_span(\n    input: EmptyModel, ctx: Context\n) -> dict[str, str]:\n    with trace_provider.get_tracer(__name__).start_as_current_span('step1'):\n        print('executed step')\n        return {\n            'foo': 'bar',\n        }\n\n\n@otel_workflow.task()\ndef your_spans_are_still_children_of_hatchet_span(\n    input: EmptyModel, ctx: Context\n) -> None:\n    with trace_provider.get_tracer(__name__).start_as_current_span('step2'):\n        raise Exception('Manually instrumented step failed failed')\n\n\n@otel_workflow.task()\ndef this_step_is_still_instrumented(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print('executed still-instrumented step')\n    return {\n        'still': 'instrumented',\n    }\n\n\n@otel_workflow.task()\ndef this_step_is_also_still_instrumented(input: EmptyModel, ctx: Context) -> None:\n    raise Exception('Still-instrumented step failed')\n\n\ndef main() -> None:\n    worker = hatchet.worker('otel-example-worker', slots=1, workflows=[otel_workflow])\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/opentelemetry_instrumentation/worker.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
