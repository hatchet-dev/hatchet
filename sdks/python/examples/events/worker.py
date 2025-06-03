from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()
EVENT_KEY = "user:create"
SECONDARY_KEY = "foobarbaz"
WILDCARD_KEY = "subscription:*"


class EventWorkflowInput(BaseModel):
    should_skip: bool


# > Event trigger
event_workflow = hatchet.workflow(
    name="EventWorkflow",
    on_events=[EVENT_KEY, SECONDARY_KEY, WILDCARD_KEY],
    input_validator=EventWorkflowInput,
)
# !!


@event_workflow.task()
def task(input: EventWorkflowInput, ctx: Context) -> None:
    print("event received")


def main() -> None:
    worker = hatchet.worker(name="EventWorker", workflows=[event_workflow])

    worker.start()


if __name__ == "__main__":
    main()
