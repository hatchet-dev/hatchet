from subprocess import Popen
from typing import Any

import pytest

from examples.bug_tests.dag_operator_task_named_like_workflow.worker import (
    task_named_like_workflow_dag,
)


@pytest.mark.timeout(20, func_only=True)
@pytest.mark.parametrize(
    "on_demand_worker",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/bug_tests/dag_operator_task_named_like_workflow/worker.py",
        ]
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_with_task_named_like_workflow(on_demand_worker: Popen[Any]) -> None:
    ref = await task_named_like_workflow_dag.aio_run(wait_for_result=False)

    assert ref.workflow_run_id

    await ref.aio_result()
