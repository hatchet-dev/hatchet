import time

from examples.durable_event.worker import (
    EVENT_KEY,
    durable_event_task,
    durable_event_task_with_filter,
    hatchet,
)

durable_event_task.run(wait_for_result=False)
durable_event_task_with_filter.run(wait_for_result=False)

print("Sleeping")
time.sleep(2)

print("Pushing event")
hatchet.event.push(
    EVENT_KEY,
    {
        "user_id": "1234",
    },
)
