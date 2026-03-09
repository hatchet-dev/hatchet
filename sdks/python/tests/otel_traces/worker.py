"""
OTel test worker — runs with HatchetInstrumentor and exposes captured spans via HTTP.

Usage:
    poetry run python tests/otel_traces/worker.py [--spans-port 8020]
"""

import argparse
import asyncio
import json
import time
from datetime import timedelta
from http.server import BaseHTTPRequestHandler, HTTPServer
from threading import Thread
from typing import Any, cast

from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter
from opentelemetry.trace import get_tracer, set_tracer_provider

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor


# -- OTel setup ----------------------------------------------------------------

span_exporter = InMemorySpanExporter()
provider = TracerProvider()
provider.add_span_processor(SimpleSpanProcessor(span_exporter))
set_tracer_provider(provider)

HatchetInstrumentor(
    tracer_provider=provider,
    enable_hatchet_otel_collector=True,
).instrument()

# -- Hatchet workflows ---------------------------------------------------------

hatchet = Hatchet(debug=True)


@hatchet.task()
def otel_simple_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    """Simple task that creates a custom child span."""
    tracer = get_tracer("otel-test")
    with tracer.start_as_current_span("custom.child.span") as span:
        span.set_attribute("test.marker", "hello")
        time.sleep(0.01)
    return {"status": "ok"}


@hatchet.task(execution_timeout=timedelta(seconds=30), retries=2)
async def otel_long_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    """Longer task for engine disconnect testing."""
    for _ in range(20):
        await asyncio.sleep(0.5)
    return {"status": "completed"}


# -- Span HTTP server ----------------------------------------------------------


def _serialize_spans() -> list[dict[str, Any]]:
    spans = span_exporter.get_finished_spans()
    result = []
    for s in spans:
        attrs = {}
        for k, v in s.attributes.items():
            attrs[k] = str(v) if not isinstance(v, (str, int, float, bool)) else v

        result.append(
            {
                "name": s.name,
                "trace_id": format(s.context.trace_id, "032x"),
                "span_id": format(s.context.span_id, "016x"),
                "parent_span_id": (
                    format(s.parent.span_id, "016x") if s.parent else None
                ),
                "attributes": attrs,
                "kind": s.kind.value if s.kind else None,
                "status_code": s.status.status_code.name if s.status else None,
            }
        )
    return result


class SpanHandler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        if self.path == "/spans":
            body = json.dumps(_serialize_spans()).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)
        else:
            self.send_response(404)
            self.end_headers()

    def do_DELETE(self) -> None:
        if self.path == "/spans":
            span_exporter.clear()
            self.send_response(200)
            self.end_headers()
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format: str, *args: Any) -> None:
        pass  # suppress request logs


def _start_span_server(port: int) -> None:
    server = HTTPServer(("0.0.0.0", port), SpanHandler)
    server.serve_forever()


# -- Main ----------------------------------------------------------------------


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--spans-port", type=int, default=8020)
    args = parser.parse_args()

    spans_port = cast(int, args.spans_port)

    # Start span server in background thread
    Thread(target=_start_span_server, args=(spans_port,), daemon=True).start()

    worker = hatchet.worker(
        "otel-e2e-test-worker",
        workflows=[otel_simple_task, otel_long_task],
    )
    worker.start()


if __name__ == "__main__":
    main()
