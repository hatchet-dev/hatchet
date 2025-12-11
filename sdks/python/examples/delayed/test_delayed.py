import asyncio
from datetime import datetime, timedelta, timezone

import pytest

from examples.delayed.worker import PrinterInput, print_printer_wf, print_schedule_wf
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_delayed_workflow(hatchet: Hatchet) -> None:
    """Test that a workflow can schedule another workflow to run in the future"""

    result = await print_schedule_wf.aio_run(
        PrinterInput(message="test delayed message")
    )

    assert result is not None

    await asyncio.sleep(20)

    since = datetime.now(tz=timezone.utc) - timedelta(minutes=1)
    runs = await print_printer_wf.aio_list_runs(since=since, limit=1)

    assert len(runs) > 0, "At least one PrintPrinterWorkflow run should exist"

    most_recent_run = runs[0]
    assert (
        most_recent_run.status == V1TaskStatus.COMPLETED
    ), f"Scheduled workflow should have completed, got {most_recent_run.status}"
