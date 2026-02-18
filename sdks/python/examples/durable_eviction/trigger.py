import time

from examples.durable_eviction.worker import (
    EVENT_KEY,
    SLEEP_TIME,
    durable_task,
    hatchet,
)

durable_task.run()
