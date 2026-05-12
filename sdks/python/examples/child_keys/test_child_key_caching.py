import asyncio
import pytest
from examples.child_keys.worker import child_key_caching_test_parent, hatchet
from uuid import uuid4
from datetime import datetime, timedelta, timezone
from hatchet_sdk import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary


async def poll_until_all_terminal(test_run_id: str) -> list[V1TaskSummary]:
    for _ in range(30):
        print("polling for runs...")
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=3),
            additional_metadata={"test_run_id": test_run_id},
        )

        if len(runs.rows) == 0:
            await asyncio.sleep(1)
            continue

        if all(
            run.status in [V1TaskStatus.COMPLETED, V1TaskStatus.FAILED]
            for run in runs.rows
        ):
            return runs.rows

        await asyncio.sleep(1)
    else:
        raise TimeoutError("Not all runs reached terminal status within timeout")


@pytest.mark.asyncio(loop_scope="session")
async def test_child_key_caching() -> None:
    test_run_id = str(uuid4())

    await child_key_caching_test_parent.aio_run(
        additional_metadata={"test_run_id": test_run_id},
        wait_for_result=False,
    )

    runs = await poll_until_all_terminal(test_run_id)

    for run in runs:
        print()
        print(run.model_dump_json(indent=2))

    assert len(runs) == 4  ## 3 children + 1 parent
