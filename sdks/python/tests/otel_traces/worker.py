"""OTel test worker with HatchetInstrumentor enabled."""

import time

from opentelemetry.trace import get_tracer

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

HatchetInstrumentor().instrument()

hatchet = Hatchet(debug=True)


@hatchet.task()
def otel_simple_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    """Simple task that creates a custom child span."""
    tracer = get_tracer("otel-test")
    with tracer.start_as_current_span("custom.child.span") as span:
        span.set_attribute("test.marker", "hello")
        time.sleep(0.01)
    return {"status": "ok"}


@hatchet.task(retries=1)
def otel_retry_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    """Task that fails on first attempt and succeeds on retry."""
    retry_count = ctx.retry_count
    if retry_count == 0:
        raise RuntimeError("intentional failure on first attempt")
    return {"status": "ok", "retry_count": str(retry_count)}


def main() -> None:
    worker = hatchet.worker(
        "otel-e2e-test-worker",
        workflows=[otel_simple_task, otel_retry_task],
    )
    worker.start()


if __name__ == "__main__":
    main()
