# Third-party integration example - requires: pip install ollama; ollama run llama2
# See: /guides/ai-agents

import json
import ollama

# > Ollama usage
def complete(messages: list[dict]) -> dict:
    resp = ollama.chat(model="llama2", messages=messages)
    content = resp.get("message", {}).get("content", "")
    tool_calls = resp.get("message", {}).get("tool_calls") or []
    return {"content": content, "tool_calls": tool_calls, "done": not tool_calls}
