# Third-party integration example - requires: pip install google-cloud-vision
# See: /guides/document-processing

from google.cloud import vision

client = vision.ImageAnnotatorClient()


# > Google Vision usage
def parse_document(content: bytes) -> str:
    image = vision.Image(content=content)
    response = client.document_text_detection(image=image)
    return response.full_text_annotation.text if response.full_text_annotation else ""
# !!
