import asyncio
from subprocess import Popen
from typing import Any

import pytest

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from tests.correct_failure_on_timeout_with_multi_concurrency.workflow import (
    TIMEOUT_SECONDS,
    InputModel,
    multiple_concurrent_cancellations_test_workflow,
)


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "tests/worker.py", "--slots", "1"]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_failure_on_timeout(
    hatchet: Hatchet, on_demand_worker: Popen[Any], test_run_id: str
) -> None:
    runs = await multiple_concurrent_cancellations_test_workflow.aio_run_many(
        [
            multiple_concurrent_cancellations_test_workflow.create_bulk_run_item(
                input=InputModel(
                    concurrency_key=test_run_id,
                ),
                additional_metadata={
                    "test_run_id": test_run_id,
                },
            )
            for _ in range(2)
        ],
        wait_for_result=False,
    )

    try:
        await asyncio.gather(*[run.aio_result() for run in runs])
    except Exception:
        pass

    results = {}
    for _ in range(int((TIMEOUT_SECONDS * 6) / 0.5)):
        results = {
            r.workflow_run_id: await hatchet.runs.aio_get(r.workflow_run_id)
            for r in runs
        }
        if all(
            run.run.status == V1TaskStatus.FAILED and len(run.task_events) > 1
            for run in results.values()
        ):
            break
        await asyncio.sleep(0.5)

    for run in results.values():
        assert run.run.status == V1TaskStatus.FAILED
        assert len(run.task_events) > 1
