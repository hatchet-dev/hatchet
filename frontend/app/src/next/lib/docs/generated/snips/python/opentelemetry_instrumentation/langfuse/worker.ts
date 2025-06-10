import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from opentelemetry.trace import get_tracer_provider\n\nfrom examples.opentelemetry_instrumentation.langfuse.client import openai\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\nfrom hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor\n\n# > Task\nHatchetInstrumentor(\n    ## Langfuse sets the global tracer provider\n    tracer_provider=get_tracer_provider(),\n).instrument()\n\nhatchet = Hatchet()\n\n\n@hatchet.task()\nasync def langfuse_task(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    ## Usage, cost, etc. of this call will be send to Langfuse\n    generation = await openai.chat.completions.create(\n        model="gpt-4o-mini",\n        messages=[\n            {"role": "system", "content": "You are a helpful assistant."},\n            {"role": "user", "content": "Where does Anna Karenina take place?"},\n        ],\n    )\n\n    location = generation.choices[0].message.content\n\n    return {\n        "location": location,\n    }\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker("langfuse-example-worker", workflows=[langfuse_task])\n    worker.start()\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/opentelemetry_instrumentation/langfuse/worker.py',
  blocks: {
    task: {
      start: 8,
      stop: 33,
    },
  },
  highlights: {},
};

export default snippet;
