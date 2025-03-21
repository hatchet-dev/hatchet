import time

from examples.waits.worker import dag_waiting_workflow, hatchet

dag_waiting_workflow.run()

time.sleep(5)

hatchet.event.push("skip_on_event:skip", {})
hatchet.event.push("wait_for_event:start", {})
