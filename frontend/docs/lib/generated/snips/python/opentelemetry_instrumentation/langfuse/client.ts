import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import base64\nimport os\n\nfrom langfuse import Langfuse  # type: ignore[import-untyped]\nfrom langfuse.openai import AsyncOpenAI  # type: ignore[import-untyped]\nfrom opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter\nfrom opentelemetry.sdk.resources import SERVICE_NAME, Resource\nfrom opentelemetry.sdk.trace import TracerProvider\nfrom opentelemetry.sdk.trace.export import BatchSpanProcessor\n\n# > Configure Langfuse\nlf = Langfuse(\n    public_key=os.getenv(\"LANGFUSE_PUBLIC_KEY\"),\n    secret_key=os.getenv(\"LANGFUSE_SECRET_KEY\"),\n    host=os.getenv(\"LANGFUSE_HOST\", \"https://app.langfuse.com\"),\n)\n\nLANGFUSE_AUTH = base64.b64encode(\n    f\"{os.getenv('LANGFUSE_PUBLIC_KEY')}:{os.getenv('LANGFUSE_SECRET_KEY')}\".encode()\n).decode()\n\nos.environ[\"OTEL_EXPORTER_OTLP_ENDPOINT\"] = (\n    os.getenv(\"LANGFUSE_HOST\", \"https://us.cloud.langfuse.com\") + \"/api/public/otel\"\n)\nos.environ[\"OTEL_EXPORTER_OTLP_HEADERS\"] = f\"Authorization=Basic {LANGFUSE_AUTH}\"\n\n# > Configure tracer provider\ntrace_provider = TracerProvider(\n    resource=Resource(\n        attributes={\n            SERVICE_NAME: os.getenv(\"HATCHET_CLIENT_OTEL_SERVICE_NAME\", \"test-service\")\n        }\n    )\n)\n\n## Add Langfuse span processor to the OpenTelemetry trace provider\ntrace_provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))\n\n# > Create an OpenAI client instrumented with Langfuse\nopenai = AsyncOpenAI(\n    api_key=os.getenv(\"OPENAI_API_KEY\"),\n)\n",
  "source": "out/python/opentelemetry_instrumentation/langfuse/client.py",
  "blocks": {
    "configure_langfuse": {
      "start": 12,
      "stop": 25
    },
    "configure_tracer_provider": {
      "start": 28,
      "stop": 37
    },
    "create_an_openai_client_instrumented_with_langfuse": {
      "start": 40,
      "stop": 42
    }
  },
  "highlights": {}
};

export default snippet;
