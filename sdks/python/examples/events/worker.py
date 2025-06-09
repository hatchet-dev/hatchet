from pydantic import BaseModel

from hatchet_sdk import Context, DefaultFilter, Hatchet

hatchet = Hatchet()


# > Event trigger
EVENT_KEY = "user:create"
SECONDARY_KEY = "foobarbaz"
WILDCARD_KEY = "subscription:*"


class EventWorkflowInput(BaseModel):
    should_skip: bool


event_workflow = hatchet.workflow(
    name="EventWorkflow",
    on_events=[EVENT_KEY, SECONDARY_KEY, WILDCARD_KEY],
    input_validator=EventWorkflowInput,
)
# !!

# > Event trigger with filter
event_workflow_with_filter = hatchet.workflow(
    name="EventWorkflow",
    on_events=[EVENT_KEY, SECONDARY_KEY, WILDCARD_KEY],
    input_validator=EventWorkflowInput,
    default_filters=[
        DefaultFilter(
            expression="true",
            scope="example-scope",
            payload={
                "main_character": "Anna",
                "supporting_character": "Stiva",
                "location": "Moscow",
            },
        )
    ],
)
# !!


@event_workflow.task()
def task(input: EventWorkflowInput, ctx: Context) -> dict[str, str]:
    print("event received")

    return dict(ctx.filter_payload)


def main() -> None:
    worker = hatchet.worker(name="EventWorker", workflows=[event_workflow])

    worker.start()


if __name__ == "__main__":
    main()
