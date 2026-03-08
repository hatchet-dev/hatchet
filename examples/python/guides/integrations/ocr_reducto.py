# Third-party integration - requires: pip install reductoai
# See: /guides/document-processing
# Reducto: parse PDFs/images to structured content, extract with schema/prompt

from reducto import Reducto

client = Reducto()


# > Reducto usage
def parse_document(content: bytes) -> str:
    upload = client.upload(file=content, extension=".pdf")
    result = client.parse.run(input=upload.file_id)
    return str(result)
