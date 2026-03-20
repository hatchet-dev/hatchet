from hatchet_sdk import Context, Hatchet
from pydantic import BaseModel

hatchet = Hatchet()


# > Step 01 Define Event Task
class EventInput(BaseModel):
    message: str
    source: str = "api"


event_wf = hatchet.workflow(
    name="EventDrivenWorkflow",
    input_validator=EventInput,
    on_events=["order:created", "user:signup"],
)


@event_wf.task()
async def process_event(input: EventInput, ctx: Context) -> dict:
    return {"processed": input.message, "source": input.source}


# !!


# > Step 02 Register Event Trigger
def push_order_event():
    """Push an event to trigger the workflow. Use the same key as on_events."""
    hatchet.event.push("order:created", {"message": "Order #1234", "source": "webhook"})


# !!


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "event-driven-worker",
        workflows=[event_wf],
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
