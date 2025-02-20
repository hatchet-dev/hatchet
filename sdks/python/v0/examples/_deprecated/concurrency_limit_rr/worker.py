import time

from dotenv import load_dotenv

from hatchet_sdk import ConcurrencyLimitStrategy, Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["concurrency-test"], schedule_timeout="10m")
class ConcurrencyDemoWorkflowRR:

    # NOTE: We're replacing the concurrency key function with a CEL expression
    #       to simplify architecture.
    #       See ../../concurrency_limit_rr/worker.py for the new implementation.
    @hatchet.concurrency(
        max_runs=1, limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN
    )
    def concurrency(self, context: Context) -> str:
        input = context.workflow_input()
        print(input)
        return f'group-{input["group"]}'

    @hatchet.step()
    def step1(self, context: Context) -> None:
        print("starting step1")
        time.sleep(2)
        print("finished step1")
        pass


workflow = ConcurrencyDemoWorkflowRR()
worker = hatchet.worker("concurrency-demo-worker-rr", max_runs=10)
worker.register_workflow(workflow)

worker.start()
