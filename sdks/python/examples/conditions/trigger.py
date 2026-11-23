import time

from examples.conditions.worker import hatchet, task_condition_workflow

task_condition_workflow.run(wait_for_result=False)

time.sleep(5)

hatchet.events.push("skip_on_event:skip", {})
hatchet.events.push("wait_for_event:start", {})
