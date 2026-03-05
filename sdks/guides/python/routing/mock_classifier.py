"""Mock classifier - no external API dependencies."""


def mock_classify(message: str) -> str:
    lower = message.lower()
    if any(w in lower for w in ("bug", "error", "help")):
        return "support"
    if any(w in lower for w in ("price", "buy", "plan")):
        return "sales"
    return "other"


def mock_reply(message: str, role: str) -> str:
    if role == "support":
        return f"[Support] I can help with that technical issue. Let me look into: {message}"
    if role == "sales":
        return f"[Sales] Great question about pricing! Here's what I can tell you about: {message}"
    return f"[General] Thanks for reaching out. Regarding: {message}"
