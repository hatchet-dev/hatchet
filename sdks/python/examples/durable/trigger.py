import time

from examples.durable.worker import (  # ephemeral_workflow,
    EVENT_KEY,
    SLEEP_TIME,
    durable_workflow,
    hatchet,
)

durable_workflow.run_no_wait()
# ephemeral_workflow.run_no_wait()

print("Sleeping")
time.sleep(SLEEP_TIME + 2)

print("Pushing event")
hatchet.event.push(EVENT_KEY, {})
