# Third-party integration example - requires: pip install openai
# See: /guides/ai-agents

import json

from openai import OpenAI
from openai.types.chat import (
    ChatCompletionAssistantMessageParam,
    ChatCompletionMessageParam,
    ChatCompletionSystemMessageParam,
    ChatCompletionToolMessageParam,
    ChatCompletionUserMessageParam,
)

from ai_agents.llm_service import ChatMessage, CompletionResult, ToolCallResult

client = OpenAI()


# > OpenAI usage
def _to_openai_message(m: ChatMessage) -> ChatCompletionMessageParam:
    if m.role == "user":
        return ChatCompletionUserMessageParam(role="user", content=m.content)
    if m.role == "assistant":
        return ChatCompletionAssistantMessageParam(role="assistant", content=m.content)
    if m.role == "system":
        return ChatCompletionSystemMessageParam(role="system", content=m.content)
    return ChatCompletionToolMessageParam(role="tool", content=m.content, tool_call_id="")


def complete(messages: list[ChatMessage]) -> CompletionResult:
    r = client.chat.completions.create(
        model="gpt-4o-mini",
        messages=[_to_openai_message(m) for m in messages],
        tool_choice="auto",
        tools=[
            {
                "type": "function",
                "function": {
                    "name": "get_weather",
                    "description": "Get weather for a location",
                    "parameters": {
                        "type": "object",
                        "properties": {"location": {"type": "string"}},
                        "required": ["location"],
                    },
                },
            }
        ],
    )
    msg = r.choices[0].message
    tool_calls = [
        ToolCallResult(name=tc.function.name, args=json.loads(tc.function.arguments or "{}"))
        for tc in (msg.tool_calls or [])
        if hasattr(tc, "function")
    ]
    return CompletionResult(content=msg.content or "", tool_calls=tool_calls, done=not tool_calls)
# !!
