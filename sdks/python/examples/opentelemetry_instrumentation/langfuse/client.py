import base64
import os

from langfuse import Langfuse  # type: ignore[import-untyped]
from langfuse.openai import AsyncOpenAI  # type: ignore[import-untyped]

# > Configure Langfuse
LANGFUSE_AUTH = base64.b64encode(
    f"{os.getenv('LANGFUSE_PUBLIC_KEY')}:{os.getenv('LANGFUSE_SECRET_KEY')}".encode()
).decode()

os.environ["OTEL_EXPORTER_OTLP_ENDPOINT"] = (
    os.getenv("LANGFUSE_HOST", "https://us.cloud.langfuse.com") + "/api/public/otel"
)
os.environ["OTEL_EXPORTER_OTLP_HEADERS"] = f"Authorization=Basic {LANGFUSE_AUTH}"

## Note: Langfuse sets the global tracer provider, so you don't need to worry about it
lf = Langfuse(
    public_key=os.getenv("LANGFUSE_PUBLIC_KEY"),
    secret_key=os.getenv("LANGFUSE_SECRET_KEY"),
    host=os.getenv("LANGFUSE_HOST", "https://app.langfuse.com"),
)
# !!

# > Create OpenAI client
openai = AsyncOpenAI(
    api_key=os.getenv("OPENAI_API_KEY"),
)
# !!
