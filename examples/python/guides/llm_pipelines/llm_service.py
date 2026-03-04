"""Encapsulated LLM service - swap MockLLMService for OpenAI/Anthropic in production.

See docs: /guides/llm-pipelines
"""

from abc import ABC, abstractmethod


class LLMService(ABC):
    """Interface for LLM generation. Implement with OpenAI, Anthropic, etc."""

    @abstractmethod
    def generate(self, prompt: str) -> dict:
        """Generate from prompt. Returns {content, valid}."""
        pass


class MockLLMService(LLMService):
    """No external API - for demos."""

    def generate(self, prompt: str) -> dict:
        return {"content": f"Generated for: {prompt[:50]}...", "valid": True}


_llm_service: LLMService | None = None


def get_llm_service() -> LLMService:
    global _llm_service
    if _llm_service is None:
        _llm_service = MockLLMService()
    return _llm_service


def set_llm_service(service: LLMService) -> None:
    global _llm_service
    _llm_service = service
