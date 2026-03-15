"""Encapsulated LLM extraction - swap MockExtractService for OpenAI/Anthropic in production.

See docs: /guides/document-processing
"""

from abc import ABC, abstractmethod


class ExtractService(ABC):
    """Interface for entity extraction from text. Implement with LLM or rules."""

    @abstractmethod
    def extract(self, text: str) -> list[str]:
        """Extract entities from parsed text."""
        pass


class MockExtractService(ExtractService):
    """No external API - returns placeholder entities for demos."""

    def extract(self, text: str) -> list[str]:
        return ["entity1", "entity2"]


_extract_service: ExtractService | None = None


def get_extract_service() -> ExtractService:
    global _extract_service
    if _extract_service is None:
        _extract_service = MockExtractService()
    return _extract_service


def set_extract_service(service: ExtractService) -> None:
    global _extract_service
    _extract_service = service
