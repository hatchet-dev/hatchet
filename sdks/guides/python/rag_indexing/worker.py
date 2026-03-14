from typing import Any

from hatchet_sdk import Context, Hatchet
from pydantic import BaseModel

from .embedding_service import get_embedding_service

hatchet = Hatchet(debug=True)


# > Step 01 Define Workflow
class DocInput(BaseModel):
    doc_id: str
    content: str


class ChunkInput(BaseModel):
    chunk: str


class QueryInput(BaseModel):
    query: str


rag_wf = hatchet.workflow(name="RAGPipeline", input_validator=DocInput)
# !!


# > Step 02 Define Ingest Task
@rag_wf.task()
async def ingest(input: DocInput, ctx: Context) -> dict[str, Any]:
    return {"doc_id": input.doc_id, "content": input.content}


# !!


# > Step 03 Chunk Task
def _chunk_content(content: str, chunk_size: int = 100) -> list[str]:
    return [content[i : i + chunk_size] for i in range(0, len(content), chunk_size)]
# !!


# > Step 04 Embed Task
@hatchet.task(name="embed-chunk", input_validator=ChunkInput)
async def embed_chunk(input: ChunkInput, ctx: Context) -> dict[str, Any]:
    embedder = get_embedding_service()
    return {"vector": embedder.embed(input.chunk)}


@rag_wf.durable_task(parents=[ingest])
async def chunk_and_embed(input: DocInput, ctx: Context) -> dict[str, Any]:
    ingested = ctx.task_output(ingest)
    chunks = [ingested["content"][i : i + 100] for i in range(0, len(ingested["content"]), 100)]
    results = await embed_chunk.aio_run_many(
        [embed_chunk.create_bulk_run_item(input=ChunkInput(chunk=c)) for c in chunks]
    )
    return {"doc_id": ingested["doc_id"], "vectors": [r["vector"] for r in results]}


# !!


# > Step 05 Query Task
@hatchet.durable_task(name="rag-query", input_validator=QueryInput)
async def query_task(input: QueryInput, ctx: Context) -> dict[str, Any]:
    result = await embed_chunk.aio_run(input=ChunkInput(chunk=input.query))
    return {"query": input.query, "vector": result["vector"], "results": []}


# !!


def main() -> None:
    # > Step 06 Run Worker
    worker = hatchet.worker(
        "rag-worker",
        workflows=[rag_wf, embed_chunk, query_task],
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
