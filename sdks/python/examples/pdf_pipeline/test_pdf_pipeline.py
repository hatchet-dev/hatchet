import base64

import pytest

from examples.pdf_pipeline.sample_pdf import make_sample_pdf
from examples.pdf_pipeline.worker import PdfInput, pdf_pipeline


@pytest.mark.asyncio(loop_scope="session")
async def test_pdf_pipeline() -> None:
    text = "Invoice from Acme Corp. Total amount due: 150 dollars."
    pdf_bytes = make_sample_pdf(text)
    content_b64 = base64.b64encode(pdf_bytes).decode()

    result = await pdf_pipeline.aio_run(
        PdfInput(filename="test-invoice.pdf", content_base64=content_b64)
    )

    assert result["extract_text"]["page_count"] == 1
    assert "Invoice" in result["extract_text"]["text"]

    assert result["classify_document"]["category"] == "invoice"

    assert result["summarize_text"]["word_count"] > 0
    assert len(result["summarize_text"]["summary"]) > 0

    keywords = result["extract_keywords"]["keywords"]
    assert len(keywords) > 0
    assert "acme" in keywords
    assert "invoice" in keywords

    assert result["format_result"]["filename"] == "test-invoice.pdf"
    assert result["format_result"]["category"] == "invoice"
    assert result["format_result"]["page_count"] == 1
    assert result["format_result"]["keywords"] == keywords
