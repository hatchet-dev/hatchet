from __future__ import annotations

import argparse
import asyncio
import os
from datetime import timedelta
from typing import cast

from hatchet_sdk import Context, DurableContext, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy
from pydantic import BaseModel, Field

WORKFLOW_PREFIX = "durable-recursive-gather-repro"


class ChildInput(BaseModel):
    path: str
    phase: str
    delay_ms: int


class ChildOutput(BaseModel):
    path: str
    phase: str


class ReproInput(BaseModel):
    depth: int = Field(default=9, ge=1, le=12)
    child_delay_ms: int = Field(default=10, ge=0, le=60_000)


class ReproOutput(BaseModel):
    child_workflow_count: int
    invocation_count: int


def _expected_child_count(depth: int) -> int:
    # There are 2**depth - 1 internal nodes. Each starts a first and follow-up
    # child, and the parent adds two outer children:
    #   2 * (2**depth - 1) + 2 == 2**(depth + 1)
    return cast(int, 2 ** (depth + 1))


hatchet = Hatchet()

# Read before the decorator runs, so it cannot come from argparse. A TTL near
# zero can evict the parent faster than it makes progress, which livelocks the
# run instead of reproducing anything.
EVICTION_TTL_MS = int(os.environ.get("REPRO2_EVICTION_TTL_MS", "1"))


@hatchet.task(
    name=f"{WORKFLOW_PREFIX}-child",
    input_validator=ChildInput,
    execution_timeout=timedelta(minutes=1),
)
async def child(params: ChildInput, _context: Context) -> ChildOutput:
    await asyncio.sleep(params.delay_ms / 1_000)
    return ChildOutput(path=params.path, phase=params.phase)


@hatchet.durable_task(
    name=f"{WORKFLOW_PREFIX}-parent",
    input_validator=ReproInput,
    execution_timeout=timedelta(minutes=20),
    retries=0,
    eviction_policy=EvictionPolicy(
        # The eviction manager checks once per second. A near-zero TTL makes the
        # parent eligible on every check while any child wait remains active.
        ttl=timedelta(milliseconds=EVICTION_TTL_MS),
        allow_capacity_eviction=True,
    ),
)
async def parent(params: ReproInput, context: DurableContext) -> ReproOutput:
    print(
        f"parent invocation={context.invocation_count} depth={params.depth} "
        f"expected_children={_expected_child_count(params.depth)}",
        flush=True,
    )

    async def run_child(path: str, phase: str) -> None:
        result = await child.aio_run(
            ChildInput(
                path=path,
                phase=phase,
                delay_ms=params.child_delay_ms,
            )
        )
        if result.path != path or result.phase != phase:
            raise RuntimeError(
                f"unexpected child result: expected {path}/{phase}, "
                f"got {result.path}/{result.phase}"
            )

    async def expand(path: str, remaining_depth: int) -> int:
        if remaining_depth == 0:
            return 0

        # A first-level child races with both recursive branches. Once all three
        # complete, this node unlocks another child while first-level children
        # from other parts of the tree may still be spawning or completing.
        _, left_count, right_count = await asyncio.gather(
            run_child(path, "first"),
            expand(f"{path}L", remaining_depth - 1),
            expand(f"{path}R", remaining_depth - 1),
        )
        await run_child(path, "follow-up")
        return 2 + left_count + right_count

    _, tree_count, _ = await asyncio.gather(
        run_child("outer-left", "outer"),
        expand("root", params.depth),
        run_child("outer-right", "outer"),
    )
    child_workflow_count = tree_count + 2
    expected_count = _expected_child_count(params.depth)
    if child_workflow_count != expected_count:
        raise RuntimeError(
            f"expected {expected_count} child workflows, got {child_workflow_count}"
        )

    return ReproOutput(
        child_workflow_count=child_workflow_count,
        invocation_count=context.invocation_count,
    )


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--attempts", type=int, default=1)
    parser.add_argument(
        "--depth",
        type=int,
        default=9,
        help="Depth 9 creates exactly 1024 child workflows.",
    )
    parser.add_argument("--child-delay-ms", type=int, default=10)
    parser.add_argument(
        "--slots",
        type=int,
        default=8,
        help=(
            "Standard worker slots for child workflows. Keeping this low makes "
            "the durable parent remain waiting across multiple eviction ticks."
        ),
    )
    parser.add_argument("--durable-slots", type=int, default=1)
    parser.add_argument(
        "--worker-startup-seconds",
        type=float,
        default=3,
        help="Time to allow workflow registration before triggering the first run.",
    )
    return parser.parse_args()


def main() -> None:
    args = _parse_args()
    worker = hatchet.worker(
        name=f"{WORKFLOW_PREFIX}-worker",
        slots=args.slots,
        durable_slots=args.durable_slots,
        workflows=[child, parent],
    )
    worker.start()


if __name__ == "__main__":
    main()
