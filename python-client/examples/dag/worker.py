from hatchet import workflow, step, Worker, Context

@workflow(on_events=["user:create"])
class MyWorkflow:
    def __init__(self):
        self.my_value = "test"

    @step()
    def step1(self, context : Context):
        print("executed step1", context.workflow_input())
        return {
            "step1": "step1",
        }

    @step()
    def step2(self, context : Context):
        print("executed step2", context.workflow_input())
        return {
            "step2": "step2",
        }

    @step(parents=["step1", "step2"])
    def step3(self, context : Context):
        print("executed step3", context.workflow_input(), context.step_output("step1"), context.step_output("step2"))
        return {
            "step3": "step3",
        }
    
    @step(parents=["step1", "step3"])
    def step4(self, context : Context):
        print("executed step4", context.workflow_input(), context.step_output("step1"), context.step_output("step3"))
        return {
            "step4": "step4",
        }

workflow = MyWorkflow()
worker = Worker('test-worker')
worker.register_workflow(workflow)

worker.start()