import time

from examples.priority.worker import high_priority_workflow, low_priority_workflow

high_priority_workflow.run_no_wait()

low_priority_workflow.run_no_wait()
high_priority_workflow.run_no_wait()
low_priority_workflow.run_no_wait()
high_priority_workflow.run_no_wait()

time.sleep(3)

low_priority_workflow.run_no_wait()
high_priority_workflow.run_no_wait()
