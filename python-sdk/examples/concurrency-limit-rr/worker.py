from hatchet_sdk import Hatchet, ConcurrencyLimitStrategy, Context
from dotenv import load_dotenv

load_dotenv()

hatchet = Hatchet(debug=True)

@hatchet.workflow(on_events=["concurrency-test"])
class ConcurrencyDemoWorkflowRR:
    @hatchet.concurrency(max_runs=1, limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN)
    def concurrency(self, context : Context) -> str:
        input = context.workflow_input()

        print(input)

        return input.get('group')

    @hatchet.step()
    def step1(self, context):
        print("starting step1")
        context.sleep(5)
        print("finished step1")
        pass

workflow = ConcurrencyDemoWorkflowRR()
worker = hatchet.worker('concurrency-demo-worker-rr', max_threads=1)
worker.register_workflow(workflow)

worker.start()