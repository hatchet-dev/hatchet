from hatchet_sdk import Hatchet, Context
from dotenv import load_dotenv
from .generate import GenerateWorkflow

load_dotenv()

hatchet = Hatchet()


def start():
    workflow = GenerateWorkflow()
    worker = hatchet.worker('test-worker')
    worker.register_workflow(workflow)

    worker.start()
