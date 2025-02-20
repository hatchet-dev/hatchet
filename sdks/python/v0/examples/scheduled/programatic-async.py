from datetime import datetime, timedelta

from dotenv import load_dotenv

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.scheduled_workflows import ScheduledWorkflows

load_dotenv()

hatchet = Hatchet()


async def create_scheduled() -> None:
    # ❓ Create
    scheduled_run = await hatchet.scheduled.aio.create(
        workflow_name="simple-workflow",
        trigger_at=datetime.now() + timedelta(seconds=10),
        input={
            "data": "simple-workflow-data",
        },
        additional_metadata={
            "customer_id": "customer-a",
        },
    )

    id = scheduled_run.metadata.id  # the id of the scheduled run trigger
    # !!

    # ❓ Delete
    await hatchet.scheduled.aio.delete(scheduled=scheduled_run.metadata.id)
    # !!

    # ❓ List
    scheduled_runs = await hatchet.scheduled.aio.list()
    # !!

    # ❓ Get
    scheduled_run = await hatchet.scheduled.aio.get(scheduled=scheduled_run.metadata.id)
    # !!
