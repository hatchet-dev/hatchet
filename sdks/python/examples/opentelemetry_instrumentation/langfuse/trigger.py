import asyncio

from langfuse import get_client  # type: ignore[import-untyped]

from examples.opentelemetry_instrumentation.langfuse.worker import langfuse_task

langfuse = get_client()


async def main() -> None:
    with langfuse.start_as_current_span(name="trigger") as span:
        result = await langfuse_task.aio_run()

        span._otel_span.set_attributes(result)


if __name__ == "__main__":
    asyncio.run(main())
