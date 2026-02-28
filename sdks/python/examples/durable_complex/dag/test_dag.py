from __future__ import annotations

import asyncio
from typing import Any

import pytest

from examples.durable_complex.dag.worker import (
    durable_dag_diamond_workflow,
    durable_dag_durable_parent_workflow,
    durable_dag_parent_failure_workflow,
    durable_dag_workflow,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus

POLL_INTERVAL = 0.2
MAX_POLLS = 150


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_dag() -> None:
    """Durable child task receives output from ephemeral parent; evicted during sleep."""
    result: dict[str, Any] = await durable_dag_workflow.aio_run()

    child_result = result.get("durable_child", result)
    assert child_result["status"] == "completed"
    assert child_result["parent_value"] == 42
    assert child_result["parent_status"] == "from_parent"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_dag_durable_parent() -> None:
    """Durable child receives output from durable parent; evicted during sleep."""
    result: dict[str, Any] = await durable_dag_durable_parent_workflow.aio_run()

    child_result = result.get("durable_child_of_durable", result)
    assert child_result["status"] == "completed"
    assert child_result["parent_value"] == 100
    assert child_result["parent_status"] == "from_durable_parent"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_dag_diamond() -> None:
    """Diamond DAG: A -> B, A -> C, B+C -> D; fan-out and fan-in; evicted during sleep."""
    result: dict[str, Any] = await durable_dag_diamond_workflow.aio_run()

    d_result = result.get("diamond_d", result)
    assert d_result["status"] == "completed"
    assert d_result["from_b"] == "done"
    assert d_result["from_c"] == "done"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_dag_parent_failure(hatchet: Hatchet) -> None:
    """When parent fails, DAG workflow fails; child does not run."""
    ref = durable_dag_parent_failure_workflow.run_no_wait()

    status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
    for _ in range(MAX_POLLS):
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        if status == V1TaskStatus.FAILED:
            break
        await asyncio.sleep(POLL_INTERVAL)
    else:
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)

    assert status == V1TaskStatus.FAILED
