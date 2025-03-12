import time

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    EmptyModel,
    Hatchet,
)

hatchet = Hatchet(debug=True)

wf = hatchet.workflow(
    name="ConcurrencyDemoWorkflowRR",
    on_events=["concurrency-test"],
    schedule_timeout="10m",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)


@wf.task()
def step1(input: EmptyModel, context: Context) -> None:
    print("starting step1")
    time.sleep(2)
    print("finished step1")
    pass


def main() -> None:
    worker = hatchet.worker("concurrency-demo-worker-rr", slots=10, workflows=[wf])

    worker.start()


if __name__ == "__main__":
    main()
