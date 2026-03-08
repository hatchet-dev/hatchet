"""Mock LLM and tools - no external API dependencies."""

from .llm_service import ChatMessage, CompletionResult, ToolCallResult

_call_count: dict[str, int] = {}


def call_llm(messages: list[ChatMessage]) -> CompletionResult:
    """Mock LLM: first call returns tool_calls, second returns final answer."""
    key = "default"
    _call_count[key] = _call_count.get(key, 0) + 1
    if _call_count[key] == 1:
        return CompletionResult(
            content="",
            tool_calls=[ToolCallResult(name="get_weather", args={"location": "SF"})],
            done=False,
        )
    return CompletionResult(
        content="It's 72°F and sunny in SF.",
        tool_calls=[],
        done=True,
    )


def run_tool(name: str, args: dict[str, str]) -> str:
    """Mock tool execution - returns canned results."""
    if name == "get_weather":
        loc = args.get("location", "unknown")
        return f"Weather in {loc}: 72°F, sunny"
    return f"Unknown tool: {name}"
