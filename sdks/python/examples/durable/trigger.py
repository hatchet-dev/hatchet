import time

from examples.durable.worker import (
    EVENT_KEY,
    SLEEP_TIME,
    durable_workflow,
    ephemeral_workflow,
    hatchet,
    AwaitedEvent,
)

durable_workflow.run_no_wait()
ephemeral_workflow.run_no_wait()

print("Sleeping")
time.sleep(SLEEP_TIME + 2)

print("Pushing event")
hatchet.event.push(EVENT_KEY, AwaitedEvent(id="123").model_dump(mode="json"))
