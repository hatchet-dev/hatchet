# Third-party integration - requires: pip install "unstructured[pdf]"
# See: /guides/document-processing
# Unstructured: open-source doc parsing for RAG, supports PDF, DOCX, images, etc.

import io
from unstructured.partition.auto import partition

# > Unstructured usage
def parse_document(content: bytes) -> str:
    elements = partition(file=io.BytesIO(content))
    return "\n\n".join(str(el) for el in elements)
