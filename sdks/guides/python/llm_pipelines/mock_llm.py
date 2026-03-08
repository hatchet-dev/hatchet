"""Mock LLM client - no external API dependencies."""

from .llm_service import GenerationResult


def generate(prompt: str) -> GenerationResult:
    """Mock: return placeholder instead of calling real LLM."""
    return GenerationResult(content=f"Generated for: {prompt[:50]}...", valid=True)


def validate(output: GenerationResult) -> bool:
    """Mock: always valid."""
    return output.valid
