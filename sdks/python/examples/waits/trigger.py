import time

from examples.waits.worker import dag_waiting_workflow, hatchet

dag_waiting_workflow.run()

time.sleep(5)

hatchet.event.push("step3:skip", {})
