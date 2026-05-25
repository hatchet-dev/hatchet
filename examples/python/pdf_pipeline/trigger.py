# > Trigger the pipeline
import base64

from examples.pdf_pipeline.sample_pdf import make_sample_pdf
from examples.pdf_pipeline.worker import PdfInput, pdf_pipeline

pdf_bytes = make_sample_pdf(
    "Invoice from Acme Corp. Total amount due: 150 dollars. Payment terms: Net 30."
)
content_b64 = base64.b64encode(pdf_bytes).decode()

result = pdf_pipeline.run(
    PdfInput(filename="sample-invoice.pdf", content_base64=content_b64)
)
print(f"Pipeline result: {result}")
