from __future__ import annotations

import asyncio
import time
from subprocess import Popen
from typing import Any

import pytest

from examples.bug_tests.subscribe_to_stream_dag.worker import (
    INTER_CHUNK_S,
    PRE_STREAM_SLEEP_S,
    STREAM_CHUNKS,
    dag_stream,
    long_stream,
)
from hatchet_sdk import EmptyModel, Hatchet
from hatchet_sdk.runnables.workflow import Workflow

pytestmark = pytest.mark.parametrize(
    "on_demand_worker",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/bug_tests/subscribe_to_stream_dag/worker.py",
        ],
    ],
    indirect=True,
)


async def collect_stream(hatchet: Hatchet, run_id: str) -> tuple[float, list[str]]:
    chunks: list[str] = []
    t0 = time.monotonic()
    async for chunk in hatchet.runs.subscribe_to_stream(run_id):
        chunks.append(chunk)
    return time.monotonic() - t0, chunks


async def subscribe_from_start(
    hatchet: Hatchet, workflow: Workflow[EmptyModel]
) -> tuple[float, list[str]]:
    ref = await workflow.aio_run(wait_for_result=False)
    elapsed, chunks = await collect_stream(hatchet, ref.workflow_run_id)
    await ref.aio_result()
    return elapsed, chunks


@pytest.mark.asyncio(loop_scope="session")
async def test_dag_subscribe_at_start_receives_every_chunk(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    elapsed, chunks = await subscribe_from_start(hatchet, dag_stream)

    assert elapsed >= 1.2
    assert chunks == STREAM_CHUNKS


@pytest.mark.asyncio(loop_scope="session")
async def test_single_task_subscribe_at_start_receives_every_chunk(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    elapsed, chunks = await subscribe_from_start(hatchet, long_stream)

    assert elapsed >= 1.2
    assert chunks == STREAM_CHUNKS


@pytest.mark.asyncio(loop_scope="session")
async def test_subscribe_while_running_receives_every_chunk(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    ref = await long_stream.aio_run(wait_for_result=False)
    await asyncio.sleep(0.8)
    elapsed, chunks = await collect_stream(hatchet, ref.workflow_run_id)
    await ref.aio_result()

    assert elapsed >= 1.2
    assert chunks == STREAM_CHUNKS


@pytest.mark.asyncio(loop_scope="session")
async def test_late_join_mid_stream_stays_open(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    ref = await long_stream.aio_run(wait_for_result=False)
    await asyncio.sleep(PRE_STREAM_SLEEP_S + INTER_CHUNK_S * 8)
    elapsed, chunks = await collect_stream(hatchet, ref.workflow_run_id)
    await ref.aio_result()

    assert elapsed >= 0.8
    assert len(chunks) > 0
    assert "".join(STREAM_CHUNKS).endswith("".join(chunks))
