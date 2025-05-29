import asyncio

from opentelemetry.trace import StatusCode

from examples.opentelemetry_instrumentation.langfuse.client import trace_provider
from examples.opentelemetry_instrumentation.langfuse.worker import langfuse_task

tracer = trace_provider.get_tracer(__name__)


async def main() -> None:
    with tracer.start_as_current_span(name="trigger") as span:
        result = await langfuse_task.aio_run()
        location = result.get("location")

        if not location:
            span.set_status(StatusCode.ERROR)
            return

        span.set_attribute("location", location)


if __name__ == "__main__":
    asyncio.run(main())
