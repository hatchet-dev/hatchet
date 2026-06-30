from examples.durable_sleep.worker import durable_sleep_task, hatchet
import time

durable_sleep_task.run(wait_for_result=False)

time.sleep(2)

hatchet.event.push("my-event", {"foo": "bar"})
