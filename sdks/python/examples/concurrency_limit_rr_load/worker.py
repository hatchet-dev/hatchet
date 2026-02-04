import random
import time

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet(debug=True)


class LoadRRInput(BaseModel):
    group: str


load_rr_workflow = hatchet.workflow(
    name="LoadRoundRobin",
    on_events=["concurrency-test"],
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
    input_validator=LoadRRInput,
)


@load_rr_workflow.on_failure_task()
def on_failure(input: LoadRRInput, context: Context) -> dict[str, str]:
    print("on_failure")
    return {"on_failure": "on_failure"}


@load_rr_workflow.task()
def step1(input: LoadRRInput, context: Context) -> dict[str, str]:
    print("starting step1")
    time.sleep(random.randint(2, 20))
    print("finished step1")
    return {"step1": "step1"}


@load_rr_workflow.task(
    retries=3,
    backoff_factor=5,
    backoff_max_seconds=60,
)
def step2(sinput: LoadRRInput, context: Context) -> dict[str, str]:
    print("starting step2")
    if random.random() < 0.5:  # 1% chance of failure
        raise Exception("Random failure in step2")
    time.sleep(2)
    print("finished step2")
    return {"step2": "step2"}


@load_rr_workflow.task()
def step3(input: LoadRRInput, context: Context) -> dict[str, str]:
    print("starting step3")
    time.sleep(0.2)
    print("finished step3")
    return {"step3": "step3"}


def main() -> None:
    worker = hatchet.worker(
        "concurrency-demo-worker-rr", slots=50, workflows=[load_rr_workflow]
    )

    worker.start()


if __name__ == "__main__":
    main()
