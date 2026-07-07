import argparse

from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabel
from pydantic import BaseModel
import asyncio
from random import random

hatchet = Hatchet()


class AffinityResult(BaseModel):
    worker_id: str


runtime_affinity_workflow = hatchet.workflow(name="runtime_affinity_workflow")


@runtime_affinity_workflow.task(retries=1)
async def validate_input(i: EmptyModel, c: Context) -> AffinityResult:
    await asyncio.sleep(1)
    return AffinityResult(worker_id=c.worker_id)


@runtime_affinity_workflow.task(parents=[validate_input], retries=1)
async def load_search_scope_meta(i: EmptyModel, c: Context) -> AffinityResult:
    if random() < 0.25 and c.attempt_number == 1:
        raise Exception("Simulated failure in load_search_scope_meta")
    await asyncio.sleep(1)
    return AffinityResult(worker_id=c.worker_id)


@runtime_affinity_workflow.task(parents=[validate_input], retries=1)
async def resolve_assessment_type(i: EmptyModel, c: Context) -> AffinityResult:
    if random() < 0.25 and c.attempt_number == 1:
        raise Exception("Simulated failure in load_search_scope_meta")
    await asyncio.sleep(1)
    return AffinityResult(worker_id=c.worker_id)


@runtime_affinity_workflow.task(
    parents=[resolve_assessment_type, load_search_scope_meta], retries=1
)
async def prepare_queries_and_exceptions(i: EmptyModel, c: Context) -> AffinityResult:
    if random() < 0.25 and c.attempt_number == 1:
        raise Exception("Simulated failure in load_search_scope_meta")
    await asyncio.sleep(1)
    return AffinityResult(worker_id=c.worker_id)


@runtime_affinity_workflow.task(
    parents=[prepare_queries_and_exceptions, load_search_scope_meta], retries=1
)
async def retrieve_context_chunks(i: EmptyModel, c: Context) -> AffinityResult:
    if random() < 0.25 and c.attempt_number == 1:
        raise Exception("Simulated failure in load_search_scope_meta")
    await asyncio.sleep(1)
    return AffinityResult(worker_id=c.worker_id)


@runtime_affinity_workflow.task(
    parents=[prepare_queries_and_exceptions, retrieve_context_chunks, validate_input],
    retries=1,
)
async def generate_llm_answer(i: EmptyModel, c: Context) -> AffinityResult:
    if random() < 0.25 and c.attempt_number == 1:
        raise Exception("Simulated failure in load_search_scope_meta")
    await asyncio.sleep(1)
    return AffinityResult(worker_id=c.worker_id)


@runtime_affinity_workflow.task(
    parents=[
        prepare_queries_and_exceptions,
        retrieve_context_chunks,
        generate_llm_answer,
    ],
    retries=1,
)
async def post_process_and_snippets(i: EmptyModel, c: Context) -> AffinityResult:
    if random() < 0.25 and c.attempt_number == 1:
        raise Exception("Simulated failure in load_search_scope_meta")
    await asyncio.sleep(1)
    return AffinityResult(worker_id=c.worker_id)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--label", type=str, required=True)
    args = parser.parse_args()

    worker = hatchet.worker(
        "runtime-affinity-worker",
        labels={"affinity": args.label},
        workflows=[runtime_affinity_workflow],
    )

    worker.start()


if __name__ == "__main__":
    main()
