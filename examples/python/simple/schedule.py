from datetime import datetime, timedelta, timezone

from examples.simple.worker import simple

# > Schedule a Task

tomorrow_noon = datetime.now(tz=timezone.utc).replace(
    hour=12, minute=0, second=0, microsecond=0
) + timedelta(days=1)

scheduled_run = simple.schedule(tomorrow_noon)

