"""Mock embedding client - no external API dependencies."""


def embed(text: str) -> list[float]:
    """Mock: return fake embedding vector."""
    return [0.1] * 64
