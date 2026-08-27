"""Driver for the durable thread/process-pool nondeterminism repro.

Per iteration:
1. trigger document_pipeline
2. while it runs, restore it whenever the engine evicts it (each restore
   replays the durable function from the top against the recorded event log)
3. after completion, replay the whole run once more via the runs API
4. record any failure containing "non-determinism"

Usage:
    poetry run python -m examples.bug_tests.durable_child_thread_nondeterminism.driver [iterations] [mode]

Modes:
    threads   (default) children use asyncio.to_thread + ProcessPoolExecutor
    nothreads same shape, no thread/process offloading at all
    evict     threads + a long-tail child so the TTL eviction fires mid-run;
              the driver restores the run instead of manually replaying
"""

import asyncio
import sys
from typing import Any

from examples.bug_tests.durable_child_thread_nondeterminism.worker import (
    PipelineInput,
    document_pipeline,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.api.task_api import TaskApi

POLL_INTERVAL_SECONDS = 0.25
PHASE_TIMEOUT_SECONDS = 180

hatchet = Hatchet()


def _restore(task_external_id: str) -> None:
    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_external_id)


async def _await_result_with_restores(
    workflow_run_id: str, result_task: "asyncio.Task[Any]", label: str
) -> tuple[str, str, int]:
    """Wait for the run to finish, restoring on every eviction.

    Returns (outcome, error_text, restore_count).
    """
    restores = 0
    elapsed = 0.0

    while not result_task.done():
        if elapsed > PHASE_TIMEOUT_SECONDS:
            result_task.cancel()
            return "timeout", "", restores

        details = await hatchet.runs.aio_get_details(workflow_run_id)
        for task_run in details.task_runs.values():
            if task_run.is_evicted:
                restores += 1
                print(
                    f"    [{label}] restoring evicted task "
                    f"{task_run.external_id} (restore #{restores})"
                )
                await asyncio.to_thread(_restore, task_run.external_id)

        await asyncio.sleep(POLL_INTERVAL_SECONDS)
        elapsed += POLL_INTERVAL_SECONDS

    try:
        result = result_task.result()
        n_docs = result.get("n_docs") if isinstance(result, dict) else None
        return "completed", f"n_docs={n_docs}", restores
    except Exception as e:  # noqa: BLE001 - we want the raw engine error text
        return "failed", str(e), restores


async def run_iteration(mode: str) -> list[tuple[str, str, str, int]]:
    outcomes: list[tuple[str, str, str, int]] = []

    pipeline_input = PipelineInput(
        n_docs=8,
        use_threads=mode in ("threads", "evict"),
        long_tail_seconds=12.0 if mode.startswith("evict") else 0.0,
    )

    ref = await document_pipeline.aio_run(pipeline_input, wait_for_result=False)
    print(f"  run {ref.workflow_run_id}")

    result_task = asyncio.ensure_future(ref.aio_result())
    outcome, detail, restores = await _await_result_with_restores(
        ref.workflow_run_id, result_task, "initial"
    )
    print(f"  initial: {outcome} ({restores} restores) {detail[:200]}")
    outcomes.append(("initial", outcome, detail, restores))

    # in evict mode the eviction+restore cycle already exercised replay
    if outcome == "completed" and mode != "evict":
        await hatchet.runs.aio_replay(ref.workflow_run_id)
        await asyncio.sleep(1)
        result_task = asyncio.ensure_future(ref.aio_result())
        outcome, detail, restores = await _await_result_with_restores(
            ref.workflow_run_id, result_task, "replay"
        )
        print(f"  replay:  {outcome} ({restores} restores) {detail[:200]}")
        outcomes.append(("replay", outcome, detail, restores))

    return outcomes


async def main() -> None:
    iterations = int(sys.argv[1]) if len(sys.argv) > 1 else 10
    mode = sys.argv[2] if len(sys.argv) > 2 else "threads"
    print(f"mode: {mode}")

    nondeterminism_hits: list[tuple[int, str, str]] = []
    other_failures: list[tuple[int, str, str]] = []

    for i in range(iterations):
        print(f"iteration {i + 1}/{iterations}")
        for phase, outcome, detail, _restores in await run_iteration(mode):
            if outcome in ("failed", "timeout"):
                if "non-determinism" in detail.lower():
                    nondeterminism_hits.append((i, phase, detail))
                else:
                    other_failures.append((i, phase, f"{outcome}: {detail}"))

    print()
    print(f"=== SUMMARY ({iterations} iterations) ===")
    print(f"nondeterminism errors: {len(nondeterminism_hits)}")
    for i, phase, detail in nondeterminism_hits:
        print(f"  iter {i} [{phase}]: {detail[:500]}")
    print(f"other failures/timeouts: {len(other_failures)}")
    for i, phase, detail in other_failures:
        print(f"  iter {i} [{phase}]: {detail[:300]}")


if __name__ == "__main__":
    asyncio.run(main())
