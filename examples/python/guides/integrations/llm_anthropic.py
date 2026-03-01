# Third-party integration example - requires: pip install anthropic
# See: /guides/ai-agents

import json
from anthropic import Anthropic

client = Anthropic()


# > Anthropic usage
def complete(messages: list[dict]) -> dict:
    resp = client.messages.create(
        model="claude-3-5-haiku-20241022",
        max_tokens=1024,
        messages=[{"role": m["role"], "content": m["content"]} for m in messages],
        tools=[{"name": "get_weather", "description": "Get weather", "input_schema": {"type": "object", "properties": {"location": {"type": "string"}}}}],
    )
    for block in resp.content:
        if block.type == "tool_use":
            return {"content": "", "tool_calls": [{"name": block.name, "args": block.input}], "done": False}
    text = "".join(b.text for b in resp.content if hasattr(b, "text"))
    return {"content": text, "tool_calls": [], "done": True}
