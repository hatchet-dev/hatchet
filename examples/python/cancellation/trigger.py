import time

from examples.cancellation.worker import cancellation_workflow, hatchet

id = cancellation_workflow.run_no_wait()

time.sleep(5)

hatchet.runs.cancel(id.workflow_run_id)
