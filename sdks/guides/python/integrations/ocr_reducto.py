# Third-party integration - requires: pip install reductoai
# See: /guides/document-processing
# Reducto: parse PDFs/images to structured content, extract with schema/prompt

from reducto import Reducto

client = Reducto()


# > Reducto usage
def parse_document(content: bytes) -> str:
    upload = client.upload.upload(file=content, extension=".pdf")
    result = client.parse.parse(input=upload.url)
    return str(result)  # or access result.blocks, result.tables, etc.
# !!
