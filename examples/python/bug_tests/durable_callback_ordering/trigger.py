from __future__ import annotations

import argparse
import asyncio
from datetime import timedelta

from examples.bug_tests.durable_callback_ordering.worker import (
    ReproInput,
    _expected_child_count,
    parent,
)


async def _run_attempts(
    *,
    attempts: int,
    depth: int,
    child_delay_ms: int,
) -> int:
    failures = 0
    for attempt in range(1, attempts + 1):
        expected_count = _expected_child_count(depth)
        print(
            f"\n=== attempt {attempt}/{attempts}: depth={depth}, "
            f"children={expected_count}, child_delay_ms={child_delay_ms} ===",
            flush=True,
        )
        try:
            result = await asyncio.wait_for(
                parent.aio_run(ReproInput(depth=depth, child_delay_ms=child_delay_ms)),
                timeout=timedelta(minutes=20).total_seconds(),
            )
        except Exception as error:
            failures += 1
            print(
                f"attempt {attempt} failed with {type(error).__name__}: {error}",
                flush=True,
            )
        else:
            print(
                f"attempt {attempt} completed: "
                f"children={result.child_workflow_count}, "
                f"final_invocation={result.invocation_count}",
                flush=True,
            )

    return failures


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


async def main() -> None:
    args = _parse_args()
    failures = await _run_attempts(
        attempts=args.attempts,
        depth=args.depth,
        child_delay_ms=args.child_delay_ms,
    )

    print(
        f"\nfinished: {failures}/{args.attempts} attempt(s) failed; "
        "inspect failures for NonDeterminismError",
        flush=True,
    )


if __name__ == "__main__":
    asyncio.run(main())
