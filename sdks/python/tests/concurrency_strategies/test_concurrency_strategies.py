import asyncio
import time
from subprocess import Popen
from typing import Any
from uuid import uuid4

import pytest

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from tests.concurrency_strategies.workflow import (
    InputModel,
    concurrency_strategy_workflow,
)


@pytest.mark.parametrize(
    "on_demand_worker",
    [(["poetry", "run", "python", "tests/worker.py", "--slots", "1"], 8002)],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_concurrency_strategy_scheduling(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    key1 = str(uuid4())
    key2 = str(uuid4())
    run = concurrency_strategy_workflow.run_no_wait(
            input=InputModel(
                key1=key1,
                key2=key2
            ),
            options=TriggerWorkflowOptions(
                additional_metadata={
                    "test_run_id": key1,
                }
            )
    )
    start = time.time()
    run.result()
    end = time.time()
    elapsed_time = end - start
    print(f"Time taken: {elapsed_time:.4f} seconds")
    assert elapsed_time < 100
