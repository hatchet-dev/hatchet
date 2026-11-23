# > Schedule a Task
from datetime import UTC, datetime

from examples.simple.worker import simple

schedule = simple.schedule(datetime(2025, 3, 14, 15, 9, 26, tzinfo=UTC))

## 👀 do something with the id
print(schedule.id)

# !!
