from hatchet_sdk import Hatchet, Context

hatchet = Hatchet()


@hatchet.workflow(on_events=["poem:create"])
class GenerateWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step()
    def step1(self, context: Context):
        print("executed step1", context.workflow_input())
        return {
            "step1": "step1",
        }

    @hatchet.step()
    def step2(self, context: Context):
        raise Exception("Step 2 failed")

    @hatchet.step(parents=["step1", "step2"])
    def step3(self, context: Context):
        print("executed step3", context.workflow_input(),
              context.step_output("step1"), context.step_output("step2"))
        return {
            "step3": "step3",
        }
