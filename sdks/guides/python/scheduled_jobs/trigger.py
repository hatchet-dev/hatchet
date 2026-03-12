from datetime import datetime, timedelta, timezone

from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)


# > Step 02 Schedule One Time
# Schedule a one-time run at a specific time.
run_at = datetime.now(tz=timezone.utc) + timedelta(hours=1)
hatchet.scheduled.create(
    workflow_name="ScheduledWorkflow",
    trigger_at=run_at,
    input={},
)
# !!
