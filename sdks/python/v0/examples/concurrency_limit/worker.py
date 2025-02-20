import time
from typing import Any

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.contracts.workflows_pb2 import ConcurrencyLimitStrategy
from hatchet_sdk.workflow import ConcurrencyExpression

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(
    on_events=["concurrency-test"],
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=5,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
)
class ConcurrencyDemoWorkflow:

    @hatchet.step()
    def step1(self, context: Context) -> dict[str, Any]:
        input = context.workflow_input()
        time.sleep(3)
        print("executed step1")
        return {"run": input["run"]}


def main() -> None:
    workflow = ConcurrencyDemoWorkflow()
    worker = hatchet.worker("concurrency-demo-worker", max_runs=10)
    worker.register_workflow(workflow)

    worker.start()


if __name__ == "__main__":
    main()
