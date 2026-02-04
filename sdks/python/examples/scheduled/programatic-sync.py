from datetime import datetime, timedelta, timezone

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.scheduled_run_status import \
    ScheduledRunStatus

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

# > Reschedule
hatchet.scheduled.update(
    scheduled_id=scheduled_run.metadata.id,
    trigger_at=datetime.now(tz=timezone.utc) + timedelta(hours=1),
)
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

# > Bulk Delete
hatchet.scheduled.bulk_delete(scheduled_ids=[id])

hatchet.scheduled.bulk_delete(
    workflow_id="workflow_id",
    statuses=[ScheduledRunStatus.SCHEDULED],
    additional_metadata={"customer_id": "customer-a"},
)
# !!

# > Bulk Reschedule
hatchet.scheduled.bulk_update(
    [
        (id, datetime.now(tz=timezone.utc) + timedelta(hours=2)),
    ]
)
# !!
