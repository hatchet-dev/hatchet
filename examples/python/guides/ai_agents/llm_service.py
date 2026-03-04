"""Encapsulated LLM service - swap MockLLMService for OpenAI/Anthropic in production.

See docs: /guides/ai-agents
"""

from abc import ABC, abstractmethod


class LLMService(ABC):
    """Interface for LLM completion. Implement with OpenAI, Anthropic, etc."""

    @abstractmethod
    def complete(self, messages: list[dict]) -> dict:
        """Complete a chat. Returns {content, tool_calls, done}."""
        pass


class MockLLMService(LLMService):
    """No external API - for local development and tests."""

    def __init__(self) -> None:
        self._call_count: dict[str, int] = {}

    def complete(self, messages: list[dict]) -> dict:
        key = "default"
        self._call_count[key] = self._call_count.get(key, 0) + 1
        if self._call_count[key] == 1:
            return {
                "content": "",
                "tool_calls": [{"name": "get_weather", "args": {"location": "SF"}}],
                "done": False,
            }
        return {"content": "It's 72°F and sunny in SF.", "tool_calls": [], "done": True}


# Default: mock. Override with getenv or DI for production.
_llm_service: LLMService | None = None


def get_llm_service() -> LLMService:
    global _llm_service
    if _llm_service is None:
        _llm_service = MockLLMService()
    return _llm_service


def set_llm_service(service: LLMService) -> None:
    global _llm_service
    _llm_service = service
