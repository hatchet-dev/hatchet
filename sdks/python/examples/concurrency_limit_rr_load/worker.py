import random
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


class LoadRR(BaseWorkflow):
    config = wf.config

    @hatchet.on_failure_step()
    def on_failure(self, context: Context) -> dict[str, str]:
        print("on_failure")
        return {"on_failure": "on_failure"}

    @hatchet.step()
    def step1(self, context: Context) -> dict[str, str]:
        print("starting step1")
        time.sleep(random.randint(2, 20))
        print("finished step1")
        return {"step1": "step1"}

    @hatchet.step(
        retries=3,
        backoff_factor=5,
        backoff_max_seconds=60,
    )
    def step2(self, context: Context) -> dict[str, str]:
        print("starting step2")
        if random.random() < 0.5:  # 1% chance of failure
            raise Exception("Random failure in step2")
        time.sleep(2)
        print("finished step2")
        return {"step2": "step2"}

    @hatchet.step()
    def step3(self, context: Context) -> dict[str, str]:
        print("starting step3")
        time.sleep(0.2)
        print("finished step3")
        return {"step3": "step3"}


def main() -> None:
    worker = hatchet.worker("concurrency-demo-worker-rr", max_runs=50)
    worker.register_workflow(LoadRR())

    worker.start()


if __name__ == "__main__":
    main()
