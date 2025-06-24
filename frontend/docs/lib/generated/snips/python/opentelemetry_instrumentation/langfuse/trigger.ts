import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\n\nfrom langfuse import get_client  # type: ignore[import-not-found]\nfrom opentelemetry.trace import StatusCode\n\nfrom examples.opentelemetry_instrumentation.langfuse.worker import langfuse_task\n\n# > Trigger task\ntracer = get_client()\n\n\nasync def main() -> None:\n    # Traces will send to Langfuse\n    # Use `_otel_tracer` to access the OpenTelemetry tracer if you need\n    # to e.g. log statuses or attributes manually.\n    with tracer._otel_tracer.start_as_current_span(name=\"trigger\") as span:\n        result = await langfuse_task.aio_run()\n        location = result.get(\"location\")\n\n        if not location:\n            span.set_status(StatusCode.ERROR)\n            return\n\n        span.set_attribute(\"location\", location)\n\n\n\nif __name__ == \"__main__\":\n    asyncio.run(main())\n",
  "source": "out/python/opentelemetry_instrumentation/langfuse/trigger.py",
  "blocks": {
    "trigger_task": {
      "start": 9,
      "stop": 26
    }
  },
  "highlights": {}
};

export default snippet;
