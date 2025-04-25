import asyncio
from random import randint
from subprocess import Popen
from typing import Any
from uuid import uuid4

import pytest

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from tests.correct_failure_on_timeout_with_multi_concurrency.workflow import (
    InputModel,
    multiple_concurrent_cancellations_test_workflow,
)


@pytest.mark.parametrize(
    "on_demand_worker",
    [(["poetry", "run", "python", "tests/worker.py", "--slots", "1"], 8002)],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_failure_on_timeout(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    test_run_id = str(uuid4())
    await multiple_concurrent_cancellations_test_workflow.aio_run_many_no_wait(
        [
            multiple_concurrent_cancellations_test_workflow.create_bulk_run_item(
                input=InputModel(
                    concurrency_key=key,
                    constant=test_run_id,
                ),
                options=TriggerWorkflowOptions(
                    additional_metadata={
                        "key": key,
                        "test_run_id": test_run_id,
                    }
                ),
            )
            for _ in range(10)
            if (key := f"{test_run_id}_key_{str(randint(1, 2))}")
        ]
    )

    await asyncio.sleep(600)
    assert False
