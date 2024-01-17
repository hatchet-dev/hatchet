from hatchet import workflow, step, Worker

@workflow(on_events=["user:create"])
class MyWorkflow:
    def __init__(self):
        self.my_value = "test"

    @step
    def step1(self, context):
        print("executed step1")
        pass

    @step
    def step2(self, context):
        print("executed step2")
        pass

workflow = MyWorkflow()
worker = Worker('test-worker')
worker.register_workflow(workflow)

worker.start()