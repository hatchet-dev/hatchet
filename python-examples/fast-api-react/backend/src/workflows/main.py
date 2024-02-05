from .hatchet import hatchet
from .generate import GenerateWorkflow, ManualTriggerWorkflow


def start():
    worker = hatchet.worker('example-worker')

    generate = GenerateWorkflow()
    trigger = ManualTriggerWorkflow()
    worker.register_workflow(generate)
    worker.register_workflow(trigger)

    worker.start()
