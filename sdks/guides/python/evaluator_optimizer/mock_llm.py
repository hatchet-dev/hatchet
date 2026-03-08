"""Mock LLM for evaluator-optimizer - no external API dependencies."""

_generate_count = 0


def mock_generate(prompt: str) -> str:
    global _generate_count
    _generate_count += 1
    if _generate_count == 1:
        return "Check out our product! Buy now!"
    return "Discover how our tool saves teams 10 hours/week. Try it free."


def mock_evaluate(draft: str) -> dict:
    if len(draft) < 40:
        return {"score": 0.4, "feedback": "Too short and pushy. Add a specific benefit and soften the CTA."}
    return {"score": 0.9, "feedback": "Clear value prop, appropriate tone."}
