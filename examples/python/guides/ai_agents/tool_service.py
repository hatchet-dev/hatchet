"""Encapsulated tool execution - swap MockToolService for real APIs in production.

See docs: /guides/ai-agents
"""

from abc import ABC, abstractmethod


class ToolService(ABC):
    """Interface for agent tool execution. Implement with your APIs."""

    @abstractmethod
    def run(self, name: str, args: dict[str, str]) -> str:
        """Execute a tool. Returns string result."""
        ...


class MockToolService(ToolService):
    """No external API - returns canned results for demos."""

    def run(self, name: str, args: dict[str, str]) -> str:
        if name == "get_weather":
            loc = args.get("location", "unknown")
            return f"Weather in {loc}: 72°F, sunny"
        return f"Unknown tool: {name}"


_tool_service: ToolService | None = None


def get_tool_service() -> ToolService:
    global _tool_service
    if _tool_service is None:
        _tool_service = MockToolService()
    return _tool_service


def set_tool_service(service: ToolService) -> None:
    global _tool_service
    _tool_service = service
