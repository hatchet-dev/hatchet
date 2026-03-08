# Third-party integration example - requires: pip install pytesseract; install Tesseract binary
# See: /guides/document-processing

import io

import pytesseract
from PIL import Image


# > Tesseract usage
def parse_document(content: bytes) -> str:
    img = Image.open(io.BytesIO(content))
    return str(pytesseract.image_to_string(img))
