from examples.priority.worker import high_priority_workflow, low_priority_workflow, control_workflow
from random import shuffle

control_workflow.run_no_wait()

funcs = [high_priority_workflow, low_priority_workflow, control_workflow]

for i in range(5):
    shuffle(funcs)

    for f in funcs:
        f.run_no_wait()
