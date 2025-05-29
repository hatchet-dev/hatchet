import base64
import os

from langfuse import Langfuse  # type: ignore[import-untyped]
from langfuse.openai import AsyncOpenAI  # type: ignore[import-untyped]
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

# > Configure Langfuse
lf = Langfuse(
    public_key=os.getenv("LANGFUSE_PUBLIC_KEY"),
    secret_key=os.getenv("LANGFUSE_SECRET_KEY"),
    host=os.getenv("LANGFUSE_HOST", "https://app.langfuse.com"),
)

LANGFUSE_AUTH = base64.b64encode(
    f"{os.getenv('LANGFUSE_PUBLIC_KEY')}:{os.getenv('LANGFUSE_SECRET_KEY')}".encode()
).decode()

os.environ["OTEL_EXPORTER_OTLP_ENDPOINT"] = (
    os.getenv("LANGFUSE_HOST", "https://us.cloud.langfuse.com") + "/api/public/otel"
)
os.environ["OTEL_EXPORTER_OTLP_HEADERS"] = f"Authorization=Basic {LANGFUSE_AUTH}"

# > Configure tracer provider
trace_provider = TracerProvider(
    resource=Resource(
        attributes={
            SERVICE_NAME: os.getenv("HATCHET_CLIENT_OTEL_SERVICE_NAME", "test-service")
        }
    )
)

## Add Langfuse span processor to the OpenTelemetry trace provider
trace_provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))

# > Create an OpenAI client instrumented with Langfuse
openai = AsyncOpenAI(
    api_key=os.getenv("OPENAI_API_KEY"),
)
