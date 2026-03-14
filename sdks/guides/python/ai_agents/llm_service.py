"""Encapsulated LLM service - swap MockLLMService for OpenAI/Anthropic in production.

See docs: /guides/ai-agents
"""

from abc import ABC, abstractmethod
from typing import Literal

from pydantic import BaseModel


class ChatMessage(BaseModel):
    role: Literal["user", "assistant", "system", "tool"]
    content: str


class ToolCallResult(BaseModel):
    name: str
    args: dict[str, str]


class CompletionResult(BaseModel):
    content: str
    tool_calls: list[ToolCallResult]
    done: bool


class LLMService(ABC):
    """Interface for LLM completion. Implement with OpenAI, Anthropic, etc."""

    @abstractmethod
    def complete(self, messages: list[ChatMessage]) -> CompletionResult:
        """Complete a chat."""
        ...


class MockLLMService(LLMService):
    """No external API - for local development and tests."""

    def __init__(self) -> None:
        self._call_count: dict[str, int] = {}

    def complete(self, messages: list[ChatMessage]) -> CompletionResult:
        key = "default"
        self._call_count[key] = self._call_count.get(key, 0) + 1
        if self._call_count[key] == 1:
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
