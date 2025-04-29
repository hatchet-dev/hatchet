from examples.concurrency_limit_rr.worker import (
    WorkflowInput,
    concurrency_limit_rr_workflow,
)
from hatchet_sdk import Hatchet

hatchet = Hatchet()

for i in range(200):
    group = "0"

    if i % 2 == 0:
        group = "1"

    concurrency_limit_rr_workflow.run(WorkflowInput(group=group))
