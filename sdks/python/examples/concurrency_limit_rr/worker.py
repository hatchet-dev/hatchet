import time

from hatchet_sdk import (
    BaseWorkflow,
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet(debug=True)

wf = hatchet.declare_workflow(
    on_events=["concurrency-test"],
    schedule_timeout="10m",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)


class ConcurrencyDemoWorkflowRR(BaseWorkflow):
    config = wf.config

    @hatchet.step()
    def step1(self, context: Context) -> None:
        print("starting step1")
        time.sleep(2)
        print("finished step1")
        pass


def main() -> None:
    worker = hatchet.worker("concurrency-demo-worker-rr", max_runs=10)
    worker.register_workflow(ConcurrencyDemoWorkflowRR())

    worker.start()


if __name__ == "__main__":
    main()
