from datetime import datetime, timedelta, timezone

from hatchet_sdk import Hatchet

hatchet = Hatchet()

# > Create
scheduled_run = hatchet.scheduled.create(
    workflow_name="simple-workflow",
    trigger_at=datetime.now(tz=timezone.utc) + timedelta(seconds=10),
    input={
        "data": "simple-workflow-data",
    },
    additional_metadata={
        "customer_id": "customer-a",
    },
)

id = scheduled_run.metadata.id  # the id of the scheduled run trigger
# !!

# > Delete
hatchet.scheduled.delete(scheduled_id=scheduled_run.metadata.id)
# !!

# > List
scheduled_runs = hatchet.scheduled.list()
# !!

# > Get
scheduled_run = hatchet.scheduled.get(scheduled_id=scheduled_run.metadata.id)
# !!
