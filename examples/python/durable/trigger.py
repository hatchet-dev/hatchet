import time

from examples.durable.worker import (
    EVENT_KEY,
    SLEEP_TIME,
    durable_workflow,
    ephemeral_workflow,
    hatchet,
)

durable_workflow.run(wait_for_result=False)
ephemeral_workflow.run(wait_for_result=False)

print("Sleeping")
time.sleep(SLEEP_TIME + 2)

print("Pushing event")
hatchet.event.push(EVENT_KEY, {})
