# > Schedule a Task
from examples.simple.worker import simple
from datetime import datetime

schedule = simple.schedule([datetime(2025, 3, 14, 15, 9, 26)])

## 👀 do something with the id
print(schedule.id)

