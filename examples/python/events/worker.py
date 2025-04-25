from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

# ❓ Event trigger
event_workflow = hatchet.workflow(name="EventWorkflow", on_events=["user:create"])
# ‼️

@event_workflow.task()
def task(input: EmptyModel, ctx: Context) -> None:
    print("event received")
