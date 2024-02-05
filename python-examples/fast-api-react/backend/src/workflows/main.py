from .hatchet import hatchet
from .generate import GenerateWorkflow


def start():
    worker = hatchet.worker('example-worker')

    generate = GenerateWorkflow()
    worker.register_workflow(generate)

    worker.start()
