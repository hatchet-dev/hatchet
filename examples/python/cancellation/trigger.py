import time

from examples.cancellation.worker import cancellation_workflow, hatchet

id = cancellation_workflow.run(wait_for_result=False)

time.sleep(5)

hatchet.runs.cancel(id.workflow_run_id)
