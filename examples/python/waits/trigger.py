import time

from examples.waits.worker import hatchet, task_condition_workflow

task_condition_workflow.run_no_wait()

time.sleep(5)

hatchet.event.push("skip_on_event:skip", {})
hatchet.event.push("wait_for_event:start", {})
