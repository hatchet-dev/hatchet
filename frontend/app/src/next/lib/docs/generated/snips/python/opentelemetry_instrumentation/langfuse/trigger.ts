import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import asyncio\n\nfrom opentelemetry.trace import StatusCode\n\nfrom examples.opentelemetry_instrumentation.langfuse.client import trace_provider\nfrom examples.opentelemetry_instrumentation.langfuse.worker import langfuse_task\n\n# > Trigger task\ntracer = trace_provider.get_tracer(__name__)\n\n\nasync def main() -> None:\n    # Traces will send to Langfuse\n    with tracer.start_as_current_span(name="trigger") as span:\n        result = await langfuse_task.aio_run()\n        location = result.get("location")\n\n        if not location:\n            span.set_status(StatusCode.ERROR)\n            return\n\n        span.set_attribute("location", location)\n\n\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
  source: 'out/python/opentelemetry_instrumentation/langfuse/trigger.py',
  blocks: {
    trigger_task: {
      start: 9,
      stop: 24,
    },
  },
  highlights: {},
};

export default snippet;
