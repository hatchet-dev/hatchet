import base64
import io

from examples.pdf_pipeline.worker import PdfInput, pdf_pipeline


def make_sample_pdf() -> bytes:
    """Create a small sample PDF for demonstration."""
    try:
        from pypdf import PdfWriter
        from pypdf.generic import (
            DecodedStreamObject,
            DictionaryObject,
            NameObject,
        )
    except ImportError:
        raise ImportError(
            "pypdf is required for this example. "
            "Install it in your Python environment before running."
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

    text = (
        "Invoice from Acme Corp. Total amount due: 150 dollars. Payment terms: Net 30."
    )
    stream = DecodedStreamObject()
    stream.set_data(f"BT /F1 12 Tf 72 720 Td ({text}) Tj ET".encode())
    page[NameObject("/Contents")] = writer._add_object(stream)

    buf = io.BytesIO()
    writer.write(buf)
    return buf.getvalue()


# > Trigger the pipeline
pdf_bytes = make_sample_pdf()
content_b64 = base64.b64encode(pdf_bytes).decode()

result = pdf_pipeline.run(
    PdfInput(filename="sample-invoice.pdf", content_base64=content_b64)
)
print(f"Pipeline result: {result}")
# !!
