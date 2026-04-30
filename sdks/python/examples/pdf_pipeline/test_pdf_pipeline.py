import base64
import io

import pytest

from examples.pdf_pipeline.worker import PdfInput, pdf_pipeline


def make_test_pdf(text: str) -> bytes:
    """Create a minimal PDF containing the given text. Uses pypdf only."""
    from pypdf import PdfWriter
    from pypdf.generic import (
        DecodedStreamObject,
        DictionaryObject,
        NameObject,
    )

    writer = PdfWriter()
    writer.add_blank_page(width=612, height=792)
    page = writer.pages[0]

    font = DictionaryObject()
    font[NameObject("/Type")] = NameObject("/Font")
    font[NameObject("/Subtype")] = NameObject("/Type1")
    font[NameObject("/BaseFont")] = NameObject("/Helvetica")
    font_ref = writer._add_object(font)

    resources = page.get("/Resources", DictionaryObject())
    if "/Font" not in resources:
        resources[NameObject("/Font")] = DictionaryObject()
    resources["/Font"][NameObject("/F1")] = font_ref
    page[NameObject("/Resources")] = resources

    stream = DecodedStreamObject()
    stream.set_data(f"BT /F1 12 Tf 72 720 Td ({text}) Tj ET".encode())
    page[NameObject("/Contents")] = writer._add_object(stream)

    buf = io.BytesIO()
    writer.write(buf)
    return buf.getvalue()


@pytest.mark.asyncio(loop_scope="session")
async def test_pdf_pipeline() -> None:
    text = "Invoice from Acme Corp. Total amount due: 150 dollars."
    pdf_bytes = make_test_pdf(text)
    content_b64 = base64.b64encode(pdf_bytes).decode()

    result = await pdf_pipeline.aio_run(
        PdfInput(filename="test-invoice.pdf", content_base64=content_b64)
    )

    assert result["extract_text"]["page_count"] == 1
    assert "Invoice" in result["extract_text"]["text"]

    assert result["classify_document"]["category"] == "invoice"

    assert result["summarize_text"]["word_count"] > 0
    assert len(result["summarize_text"]["summary"]) > 0

    assert result["format_result"]["filename"] == "test-invoice.pdf"
    assert result["format_result"]["category"] == "invoice"
    assert result["format_result"]["page_count"] == 1
