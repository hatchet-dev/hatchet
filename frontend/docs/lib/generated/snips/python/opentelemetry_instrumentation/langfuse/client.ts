import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import base64\nimport os\n\nfrom langfuse import Langfuse  # type: ignore\nfrom langfuse.openai import AsyncOpenAI  # type: ignore\n\n# > Configure Langfuse\nLANGFUSE_AUTH = base64.b64encode(\n    f\"{os.getenv('LANGFUSE_PUBLIC_KEY')}:{os.getenv('LANGFUSE_SECRET_KEY')}\".encode()\n).decode()\n\nos.environ[\"OTEL_EXPORTER_OTLP_ENDPOINT\"] = (\n    os.getenv(\"LANGFUSE_HOST\", \"https://us.cloud.langfuse.com\") + \"/api/public/otel\"\n)\nos.environ[\"OTEL_EXPORTER_OTLP_HEADERS\"] = f\"Authorization=Basic {LANGFUSE_AUTH}\"\n\n## Note: Langfuse sets the global tracer provider, so you don't need to worry about it\nlf = Langfuse(\n    public_key=os.getenv(\"LANGFUSE_PUBLIC_KEY\"),\n    secret_key=os.getenv(\"LANGFUSE_SECRET_KEY\"),\n    host=os.getenv(\"LANGFUSE_HOST\", \"https://app.langfuse.com\"),\n)\n\n# > Create OpenAI client\nopenai = AsyncOpenAI(\n    api_key=os.getenv(\"OPENAI_API_KEY\"),\n)\n",
  "source": "out/python/opentelemetry_instrumentation/langfuse/client.py",
  "blocks": {
    "configure_langfuse": {
      "start": 8,
      "stop": 22
    },
    "create_openai_client": {
      "start": 25,
      "stop": 27
    }
  },
  "highlights": {}
};

export default snippet;
