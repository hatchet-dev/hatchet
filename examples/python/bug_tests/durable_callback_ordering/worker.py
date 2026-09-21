from __future__ import annotations

import asyncio
from datetime import timedelta

from hatchet_sdk import Context, DurableContext, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy
from pydantic import BaseModel, Field

WORKFLOW_PREFIX = "durable-callback-ordering"

hatchet = Hatchet()


class LeafInput(BaseModel):
    mid: int
    branch: int
    delay_ms: int
    generation: int


class LeafOutput(BaseModel):
    mid: int
    branch: int
    generation: int


class MidInput(BaseModel):
    mid: int
    branches: int = Field(ge=2, le=100)
    child_delay_ms: int = Field(ge=1_100, le=60_000)
    delay_step_ms: int = Field(ge=0, le=1_000)


class MidOutput(BaseModel):
    mid: int
    completed_branches: list[int]
    invocation_count: int


class RootInput(BaseModel):
    durables: int = Field(default=4, ge=2, le=32)
    branches: int = Field(default=8, ge=2, le=100)
    child_delay_ms: int = Field(default=1_500, ge=1_100, le=60_000)
    delay_step_ms: int = Field(default=3, ge=0, le=1_000)


class RootOutput(BaseModel):
    root_invocation_count: int
    mid_invocation_counts: list[int]
    completed_mids: list[int]


@hatchet.task(
    name=f"{WORKFLOW_PREFIX}-leaf",
    input_validator=LeafInput,
    execution_timeout=timedelta(minutes=1),
)
async def callback_ordering_leaf(params: LeafInput, _context: Context) -> LeafOutput:
    await asyncio.sleep(params.delay_ms / 1_000)
    return LeafOutput(
        mid=params.mid, branch=params.branch, generation=params.generation
    )


@hatchet.durable_task(
    name=f"{WORKFLOW_PREFIX}-mid",
    input_validator=MidInput,
    execution_timeout=timedelta(minutes=5),
    retries=0,
    eviction_policy=EvictionPolicy(
        ttl=timedelta(milliseconds=250),
        allow_capacity_eviction=True,
    ),
)
async def callback_ordering_mid(params: MidInput, context: DurableContext) -> MidOutput:
    # Staggered first-generation children complete out of spawn order, so each
    # branch's second-generation spawn is emitted in completion order. Replays
    # after eviction must re-deliver those completions in the recorded order or
    # the re-emitted spawn sequence diverges from the event log.
    async def branch(branch_index: int) -> int:
        first_delay_ms = (
            params.child_delay_ms
            + (params.branches - branch_index - 1) * params.delay_step_ms
        )
        first = await callback_ordering_leaf.aio_run(
            LeafInput(
                mid=params.mid,
                branch=branch_index,
                delay_ms=first_delay_ms,
                generation=1,
            )
        )
        second = await callback_ordering_leaf.aio_run(
            LeafInput(
                mid=params.mid,
                branch=first.branch,
                delay_ms=params.child_delay_ms,
                generation=2,
            )
        )
        return second.branch

    completed = await asyncio.gather(
        *(branch(branch_index) for branch_index in range(params.branches))
    )
    return MidOutput(
        mid=params.mid,
        completed_branches=list(completed),
        invocation_count=context.invocation_count,
    )


@hatchet.durable_task(
    name=f"{WORKFLOW_PREFIX}-root",
    input_validator=RootInput,
    execution_timeout=timedelta(minutes=10),
    retries=0,
    eviction_policy=EvictionPolicy(
        ttl=timedelta(milliseconds=250),
        allow_capacity_eviction=True,
    ),
)
async def callback_ordering_root(
    params: RootInput, context: DurableContext
) -> RootOutput:
    results = await asyncio.gather(
        *(
            callback_ordering_mid.aio_run(
                MidInput(
                    mid=mid_index,
                    branches=params.branches,
                    child_delay_ms=params.child_delay_ms,
                    delay_step_ms=params.delay_step_ms,
                )
            )
            for mid_index in range(params.durables)
        )
    )
    return RootOutput(
        root_invocation_count=context.invocation_count,
        mid_invocation_counts=[item.invocation_count for item in results],
        completed_mids=[item.mid for item in results],
    )
