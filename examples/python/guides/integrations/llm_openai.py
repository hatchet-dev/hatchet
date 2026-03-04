# Third-party integration example - requires: pip install openai
# See: /guides/ai-agents

import json

from openai import OpenAI

client = OpenAI()


# > OpenAI usage
def complete(messages: list[dict]) -> dict:
    r = client.chat.completions.create(
        model="gpt-4o-mini",
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
