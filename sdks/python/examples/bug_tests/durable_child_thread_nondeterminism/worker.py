"""Repro for reported nondeterminism errors on the virtualized (evictable)
durable engine when non-durable child tasks use asyncio.to_thread and
ProcessPoolExecutor.

Shape of the reported workload (document pipeline):
- durable parent runs several per-document flows concurrently via asyncio.gather
- each flow spawns a non-durable child, does thread-offloaded work, then spawns
  a second child (so later spawn ordering depends on thread/child timing)
- children offload to asyncio.to_thread and a ProcessPoolExecutor
- the parent has an eviction policy, so it can be evicted mid-run and replayed
"""

import asyncio
import hashlib
from concurrent.futures import ProcessPoolExecutor
from datetime import timedelta
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, DurableContext, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet()

EVICTION_TTL_SECONDS = 5

_process_pool: ProcessPoolExecutor | None = None


def _get_process_pool() -> ProcessPoolExecutor:
    # NOTE: lazy so importing this module (e.g. from the driver) does not fork pools
    global _process_pool
    if _process_pool is None:
        _process_pool = ProcessPoolExecutor(max_workers=2)
    return _process_pool


def _cpu_work(seed: int, rounds: int = 20_000) -> str:
    h = hashlib.sha256(str(seed).encode())
    for _ in range(rounds):
        h = hashlib.sha256(h.digest())
    return h.hexdigest()


class DocStageInput(BaseModel):
    doc_id: int
    stage: str
    sleep_seconds: float
    use_threads: bool = True


class PipelineInput(BaseModel):
    n_docs: int = 8
    use_threads: bool = True
    # sleep for doc 0's extract child; set above the eviction TTL to force a
    # TTL eviction while the parent is mid-gather
    long_tail_seconds: float = 0.0


@hatchet.task(input_validator=DocStageInput)
async def doc_stage_child(input: DocStageInput, ctx: Context) -> dict[str, Any]:
    thread_digest = ""
    process_digest = ""

    if input.use_threads:
        thread_digest = await asyncio.to_thread(_cpu_work, input.doc_id)

        loop = asyncio.get_running_loop()
        process_digest = await loop.run_in_executor(
            _get_process_pool(), _cpu_work, input.doc_id + 1000
        )

    await asyncio.sleep(input.sleep_seconds)

    return {
        "doc_id": input.doc_id,
        "stage": input.stage,
        "thread_digest": thread_digest[:8],
        "process_digest": process_digest[:8],
    }


@hatchet.durable_task(
    input_validator=PipelineInput,
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EvictionPolicy(
        ttl=timedelta(seconds=EVICTION_TTL_SECONDS),
        allow_capacity_eviction=True,
    ),
)
async def document_pipeline(
    input: PipelineInput, ctx: DurableContext
) -> dict[str, Any]:
    async def per_doc(doc_id: int) -> dict[str, Any]:
        extract_sleep = 1.0 + (doc_id % 4)
        if doc_id == 0 and input.long_tail_seconds:
            extract_sleep = input.long_tail_seconds

        extract = await doc_stage_child.aio_run(
            input=DocStageInput(
                doc_id=doc_id,
                stage="extract",
                # variable child runtimes so completion order differs between
                # the first invocation and post-eviction replays
                sleep_seconds=extract_sleep,
                use_threads=input.use_threads,
            )
        )

        # thread-offloaded work between the two durable spawns, mirroring the
        # reported asyncio.to_thread usage inside the pipeline
        if input.use_threads:
            await asyncio.to_thread(_cpu_work, doc_id)

        # NOTE: all child inputs are deterministic functions of doc_id, so the
        # idempotency hash for a given logical spawn is identical across
        # replays; only scheduling/ordering can differ
        classify = await doc_stage_child.aio_run(
            input=DocStageInput(
                doc_id=doc_id,
                stage="classify",
                sleep_seconds=((doc_id * 7) % 5) / 4,
                use_threads=input.use_threads,
            )
        )

        return {"extract": extract, "classify": classify}

    results = await asyncio.gather(*(per_doc(i) for i in range(input.n_docs)))

    return {"n_docs": len(results), "results": results}


def main() -> None:
    worker = hatchet.worker(
        "durable-thread-nondeterminism-worker",
        workflows=[document_pipeline, doc_stage_child],
    )
    worker.start()


if __name__ == "__main__":
    main()
