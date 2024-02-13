from hatchet_sdk import Hatchet, Context
from dotenv import load_dotenv

load_dotenv()

hatchet = Hatchet(debug=True)

@hatchet.workflow(on_events=["user:create"])
class MyWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step()
    def step1(self, context: Context):
        test = context.overrides("test", "test")
        test2 = context.overrides("test2", 100)
        test3 = context.overrides("test3", None)

        print(test)
        print(test2)

        print("executed step1")
        pass

    @hatchet.step(parents=["step1"],timeout='4s')
    def step2(self, context):
        print("started step2")
        context.sleep(1)
        print("finished step2")

workflow = MyWorkflow()
worker = hatchet.worker('test-worker', max_threads=4)
worker.register_workflow(workflow)

worker.start()