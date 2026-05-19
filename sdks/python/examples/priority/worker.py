import time

from hatchet_sdk import Context, EmptyModel, Hatchet, Priority

hatchet = Hatchet()

# > Default priority
DEFAULT_PRIORITY = Priority.LOW
SLEEP_TIME = 0.25

priority_workflow = hatchet.workflow(
    name="PriorityWorkflow",
    default_priority=DEFAULT_PRIORITY,
)
# !!


@priority_workflow.task()
def priority_task(input: EmptyModel, ctx: Context) -> None:
    print("Priority:", ctx.priority)
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
