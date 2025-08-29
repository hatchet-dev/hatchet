import asyncio

import pytest

from examples.logger.workflow import N, logging_workflow
from hatchet_sdk import ClientConfig, Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_log_capture(hatchet: Hatchet) -> None:
    ref = await logging_workflow.aio_run_no_wait()

    result = await ref.aio_result()

    assert result["root_logger"]["status"] == "success"

    run = await hatchet.runs.aio_get(ref.workflow_run_id)

    subtask_ids = [t.metadata.id for t in run.tasks]

    await asyncio.sleep(hatchet.config.log_flush_interval_seconds * 7)

    for subtask_id in subtask_ids:
        logs = await hatchet.logs.aio_list(task_run_id=subtask_id)

        assert logs.rows
        assert len(logs.rows) == N * 2
