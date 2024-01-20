from hatchet import Hatchet

hatchet = Hatchet()

@hatchet.workflow(on_events=["user:create"])
class MyWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step()
    def step1(self, context):
        print("executed step1")
        pass

    @hatchet.step(parents=["step1"])
    def step2(self, context):
        print("executed step2")
        pass

workflow = MyWorkflow()
worker = hatchet.worker('test-worker')
worker.register_workflow(workflow)

worker.start()