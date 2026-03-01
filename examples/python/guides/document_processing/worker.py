from typing import Any

from hatchet_sdk import Context, Hatchet
from pydantic import BaseModel

try:
    from .ocr_service import get_ocr_service
    from .llm_extract_service import get_extract_service
except ImportError:
    from ocr_service import get_ocr_service
    from llm_extract_service import get_extract_service

hatchet = Hatchet(debug=True)


# > Step 01 Define DAG
class DocInput(BaseModel):
    doc_id: str
    content: bytes = b""


doc_wf = hatchet.workflow(name="DocumentPipeline", input_validator=DocInput)


@doc_wf.task()
async def ingest(input: DocInput, ctx: Context) -> dict[str, Any]:
    return {"doc_id": input.doc_id, "content": input.content}




# > Step 02 Parse Stage
@doc_wf.task(parents=[ingest])
async def parse(input: DocInput, ctx: Context) -> dict[str, Any]:
    ingested = ctx.task_output(ingest)
    text = get_ocr_service().parse(ingested["content"])
    return {"doc_id": input.doc_id, "text": text}




# > Step 03 Extract Stage
@doc_wf.task(parents=[parse])
async def extract(input: DocInput, ctx: Context) -> dict[str, Any]:
    parsed = ctx.task_output(parse)
    entities = get_extract_service().extract(parsed["text"])
    return {"doc_id": parsed["doc_id"], "entities": entities}




def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "document-worker",
        workflows=[doc_wf],
    )
    worker.start()


if __name__ == "__main__":
    main()
