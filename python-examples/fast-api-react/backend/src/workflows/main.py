from .hatchet import hatchet
from .generate import GenerateWorkflow


def start():
    workflow = GenerateWorkflow()
    worker = hatchet.worker('test-worker')
    worker.register_workflow(workflow)

    worker.start()
