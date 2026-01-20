import os
from typing import cast

from dotenv import load_dotenv

load_dotenv()  # Load .env file from current directory or parents

from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import NoOpTracerProvider

trace_provider: TracerProvider | NoOpTracerProvider

if os.getenv("CI", "false") == "true":
    trace_provider = NoOpTracerProvider()
else:
    resource = Resource(
        attributes={
            SERVICE_NAME: os.getenv("HATCHET_CLIENT_OTEL_SERVICE_NAME", "test-service")
        }
    )
    print(os.getenv("HATCHET_CLIENT_OTEL_EXPORTER_OTLP_HEADERS"))

    headers = dict(
        [
            cast(
                tuple[str, str],
                tuple(
                    os.getenv(
                        "HATCHET_CLIENT_OTEL_EXPORTER_OTLP_HEADERS", "foo=bar"
                    ).split("=")
                ),
            )
        ]
    )

    processor = BatchSpanProcessor(
        OTLPSpanExporter(
            endpoint=os.getenv(
                "HATCHET_CLIENT_OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318/v1/traces"
            ),
            headers=headers,
        ),
    )

    trace_provider = TracerProvider(resource=resource)

    trace_provider.add_span_processor(processor)
