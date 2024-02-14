from hatchet_sdk import Hatchet, Context
from dotenv import load_dotenv

load_dotenv()

hatchet = Hatchet()

@hatchet.workflow(on_events=["user:create"])
class MyWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step()
    def step1(self, context : Context):
        context.overrides("test", "test")

        print("executed step1", context.workflow_input())
        return {
            "step1": "step1",
        }

    @hatchet.step()
    def step2(self, context : Context):
        print("executed step2", context.workflow_input())
        return {
            "step2": "step2",
        }

    @hatchet.step(parents=["step1", "step2"])
    def step3(self, context : Context):
        print("executed step3", context.workflow_input(), context.step_output("step1"), context.step_output("step2"))
        return {
            "step3": "step3",
        }
    
    @hatchet.step(parents=["step1", "step3"])
    def step4(self, context : Context):
        print("executed step4", context.workflow_input(), context.step_output("step1"), context.step_output("step3"))
        return {
            "step4": "step4",
        }

workflow = MyWorkflow()
worker = hatchet.worker('test-worker')
worker.register_workflow(workflow)

worker.start()