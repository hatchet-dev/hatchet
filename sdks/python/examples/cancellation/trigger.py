# > Cancelling a run
import time

from examples.cancellation.worker import cancellation_workflow, hatchet

ref = cancellation_workflow.run_no_wait()

time.sleep(5)

## Cancel by run ID
hatchet.runs.cancel(ref.workflow_run_id)
# !!
