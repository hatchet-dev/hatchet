"""Mock OCR/parser - no external dependencies."""


def parse_document(content: bytes) -> str:
    """Mock: return placeholder text instead of real OCR."""
    return f"Parsed text from {len(content)} bytes"
