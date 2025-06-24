import asyncio

from langfuse import get_client  # type: ignore[import-not-found]
from opentelemetry.trace import StatusCode

from examples.opentelemetry_instrumentation.langfuse.worker import langfuse_task

# > Trigger task
tracer = get_client()


async def main() -> None:
    # Traces will send to Langfuse
    # Use `_otel_tracer` to access the OpenTelemetry tracer if you need
    # to e.g. log statuses or attributes manually.
    with tracer._otel_tracer.start_as_current_span(name="trigger") as span:
        result = await langfuse_task.aio_run()
        location = result.get("location")

        if not location:
            span.set_status(StatusCode.ERROR)
            return

        span.set_attribute("location", location)



if __name__ == "__main__":
    asyncio.run(main())
