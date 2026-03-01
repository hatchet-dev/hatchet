"""Mock LLM client - no external API dependencies."""


def generate(prompt: str) -> dict:
    """Mock: return placeholder instead of calling real LLM."""
    return {"content": f"Generated for: {prompt[:50]}...", "valid": True}


def validate(output: dict) -> bool:
    """Mock: always valid."""
    return output.get("valid", False)
