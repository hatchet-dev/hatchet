import asyncio
import contextlib
from subprocess import Popen
from typing import Any

import pytest

from examples.test_utils import poll_until
from hatchet_sdk import Hatchet
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

    with contextlib.suppress(Exception):
        await asyncio.gather(*[run.aio_result() for run in runs])

    async def wait_for_timeout_failure(run_id: str) -> None:
        await poll_until(
            lambda: hatchet.runs.aio_get(run_id),
            lambda details: details.run.status == "FAILED"
            and len(details.task_events) > 1,
            description=f"run {run_id} to fail with its timeout events",
            timeout=TIMEOUT_SECONDS * 6,
        )

    await asyncio.gather(*[wait_for_timeout_failure(r.workflow_run_id) for r in runs])
