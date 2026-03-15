# Third-party integration - requires: pip install groq
# See: /guides/ai-agents

import json

from groq import Groq

client = Groq()


# > Groq usage
def complete(messages: list[dict]) -> dict:
    r = client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=messages,
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
        {"name": tc.function.name, "args": json.loads(tc.function.arguments or "{}")}
        for tc in (msg.tool_calls or [])
    ]
    return {"content": msg.content or "", "tool_calls": tool_calls, "done": not tool_calls}
# !!
