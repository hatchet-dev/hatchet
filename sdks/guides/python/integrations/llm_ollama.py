# Third-party integration example - requires: pip install ollama; ollama run llama2
# See: /guides/ai-agents

import ollama
from ollama import Message

from ai_agents.llm_service import ChatMessage, CompletionResult, ToolCallResult


# > Ollama usage
def complete(messages: list[ChatMessage]) -> CompletionResult:
    ollama_messages: list[Message] = [
        Message(role=m.role, content=m.content) for m in messages
    ]
    resp = ollama.chat(model="llama2", messages=ollama_messages)
    msg = resp.get("message", {})
    raw_calls = msg.get("tool_calls") or []
    tool_calls = [
        ToolCallResult(name=tc["function"]["name"], args=tc["function"].get("arguments", {}))
        for tc in raw_calls
    ]
    return CompletionResult(
        content=msg.get("content", ""),
        tool_calls=tool_calls,
        done=not tool_calls,
    )
# !!
