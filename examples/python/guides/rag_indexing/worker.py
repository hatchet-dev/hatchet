from typing import Any

from hatchet_sdk import Context, Hatchet
from pydantic import BaseModel

try:
    from .embedding_service import get_embedding_service
except ImportError:
    from embedding_service import get_embedding_service

hatchet = Hatchet(debug=True)


# > Step 01 Define Ingest Task
class DocInput(BaseModel):
    doc_id: str
    content: str


rag_wf = hatchet.workflow(name="RAGPipeline", input_validator=DocInput)


@rag_wf.task()
async def ingest(input: DocInput, ctx: Context) -> dict[str, Any]:
    return {"doc_id": input.doc_id, "content": input.content}




# > Step 02 Chunk Task
def _chunk_content(content: str, chunk_size: int = 100) -> list[str]:
    return [content[i : i + chunk_size] for i in range(0, len(content), chunk_size)]


# > Step 03 Embed Task
@rag_wf.task(parents=[ingest])
async def chunk_and_embed(input: DocInput, ctx: Context) -> dict[str, Any]:
    ingested = ctx.task_output(ingest)
    chunks = [ingested["content"][i : i + 100] for i in range(0, len(ingested["content"]), 100)]
    embedder = get_embedding_service()
    vectors = [embedder.embed(c) for c in chunks]
    return {"doc_id": ingested["doc_id"], "vectors": vectors}




def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "rag-worker",
        workflows=[rag_wf],
    )
    worker.start()


if __name__ == "__main__":
    main()
