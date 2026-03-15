"""Mock LLM for parallelization - no external API dependencies."""


def mock_generate_content(message: str) -> str:
    return f"Here is a helpful response to: {message}"


def mock_safety_check(message: str) -> dict:
    if "unsafe" in message.lower():
        return {"safe": False, "reason": "Content flagged as potentially unsafe."}
    return {"safe": True, "reason": "Content is appropriate."}


def mock_evaluate(content: str) -> dict:
    score = 0.85 if len(content) > 20 else 0.3
    return {"score": score, "approved": score >= 0.7}
