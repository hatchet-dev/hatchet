from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class EventWorkflowInput(BaseModel):
    should_skip: bool


hatchet = Hatchet()

# > Event trigger
event_workflow = hatchet.workflow(
    name="EventWorkflow",
    on_events=["user:create"],
    event_filter_expression="input.should_skip == true",
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
