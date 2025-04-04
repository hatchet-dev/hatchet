import time

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    EmptyModel,
    Hatchet,
)

hatchet = Hatchet(debug=True)

DEFAULT_PRIORITY = 2
SLEEP_TIME = 0.25

priority_workflow = hatchet.workflow(
    name="PriorityWorkflow",
    default_priority=DEFAULT_PRIORITY,
    concurrency=ConcurrencyExpression(
        max_runs=1,
        expression="'true'",
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)


@priority_workflow.task()
def priority_task(input: EmptyModel, ctx: Context) -> None:
    time.sleep(SLEEP_TIME)


def main() -> None:
    worker = hatchet.worker(
        "priority-worker",
        slots=1,
        workflows=[priority_workflow],
    )

    worker.start()


if __name__ == "__main__":
    main()
