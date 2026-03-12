"""Encapsulated OCR service - swap MockOCRService for Tesseract/Google Vision in production.

See docs: /guides/document-processing
"""

from abc import ABC, abstractmethod


class OCRService(ABC):
    """Interface for document parsing. Implement with Tesseract, Google Vision, etc."""

    @abstractmethod
    def parse(self, content: bytes) -> str:
        """Convert raw bytes (image/PDF) to text."""
        pass


class MockOCRService(OCRService):
    """No external API - returns placeholder for demos."""

    def parse(self, content: bytes) -> str:
        return f"Parsed text from {len(content)} bytes"


_ocr_service: OCRService | None = None


def get_ocr_service() -> OCRService:
    global _ocr_service
    if _ocr_service is None:
        _ocr_service = MockOCRService()
    return _ocr_service


def set_ocr_service(service: OCRService) -> None:
    global _ocr_service
    _ocr_service = service
