"""Encapsulated embedding service - swap MockEmbeddingService for OpenAI/Cohere in production.

See docs: /guides/rag-and-indexing
"""

from abc import ABC, abstractmethod


class EmbeddingService(ABC):
    """Interface for text embeddings. Implement with OpenAI, Cohere, etc."""

    @abstractmethod
    def embed(self, text: str) -> list[float]:
        """Convert text to embedding vector."""
        pass


class MockEmbeddingService(EmbeddingService):
    """No external API - returns placeholder vectors for demos."""

    def __init__(self, dim: int = 64) -> None:
        self.dim = dim

    def embed(self, text: str) -> list[float]:
        return [0.1] * self.dim


_embedding_service: EmbeddingService | None = None


def get_embedding_service() -> EmbeddingService:
    global _embedding_service
    if _embedding_service is None:
        _embedding_service = MockEmbeddingService()
    return _embedding_service


def set_embedding_service(service: EmbeddingService) -> None:
    global _embedding_service
    _embedding_service = service
