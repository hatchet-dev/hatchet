import time

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

high_priority_workflow = hatchet.workflow(
    name="HighPriorityWorkflow", default_priority=3, on_events=["high_prio:1"]
)
low_priority_workflow = hatchet.workflow(name="LowPriorityWorkflow", default_priority=1, on_events=["low_prio:1"])

SLEEP_TIME = 5


@high_priority_workflow.task()
def high_prio_task(input: EmptyModel, ctx: Context) -> None:
    time.sleep(SLEEP_TIME)


@low_priority_workflow.task()
def low_prio_task(input: EmptyModel, ctx: Context) -> None:
    time.sleep(SLEEP_TIME)


def main() -> None:
    worker = hatchet.worker(
        "priority-worker",
        slots=1,
        workflows=[low_priority_workflow, high_priority_workflow],
    )

    worker.start()


if __name__ == "__main__":
    main()
