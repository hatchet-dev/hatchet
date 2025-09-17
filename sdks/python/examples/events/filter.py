from examples.events.worker import EVENT_KEY, event_workflow
from hatchet_sdk import Hatchet, PushEventOptions

hatchet = Hatchet()

# > Create a filter
hatchet.filters.create(
    workflow_id=event_workflow.id,
    expression="input.should_skip == false",
    scope="foobarbaz",
    payload={
        "main_character": "Anna",
        "supporting_character": "Stiva",
        "location": "Moscow",
    },
)
# !!

# > Skip a run
hatchet.event.push(
    event_key=EVENT_KEY,
    payload={
        "should_skip": True,
    },
    options=PushEventOptions(
        scope="foobarbaz",
    ),
)
# !!

# > Trigger a run
hatchet.event.push(
    event_key=EVENT_KEY,
    payload={
        "should_skip": False,
    },
    options=PushEventOptions(
        scope="foobarbaz",
    ),
)
# !!
