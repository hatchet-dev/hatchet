from dotenv import load_dotenv

from hatchet_sdk import Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["concurrency-test"])
class ConcurrencyDemoWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.concurrency(max_runs=5)
    def concurrency(self, context) -> str:
        return "concurrency-key"

    @hatchet.step()
    def step1(self, context):
        print("executed step1")
        pass

    @hatchet.step(parents=["step1"], timeout="4s")
    def step2(self, context):
        print("started step2")
        context.sleep(1)
        print("finished step2")


workflow = ConcurrencyDemoWorkflow()
worker = hatchet.worker("concurrency-demo-worker", max_runs=4)
worker.register_workflow(workflow)

worker.start()
