from examples.events.worker import EVENT_KEY, event_workflow
from hatchet_sdk import Hatchet

hatchet = Hatchet()

# > Create a filter
hatchet.filters.create(
    workflow_id=event_workflow.id,
    expression="input.should_skip == false",
    # the scope groups filters: only events pushed with a matching
    # scope are evaluated against this filter. in a real app, this is
    # often an id, e.g. a customer, user, or organization id
    scope="foobarbaz",
    payload={
        "main_character": "Anna",
        "supporting_character": "Stiva",
        "location": "Moscow",
    },
)

# > Skip a run
hatchet.event.push(
    event_key=EVENT_KEY,
    payload={
        "should_skip": True,
    },
    scope="foobarbaz",
)

# > Trigger a run
hatchet.event.push(
    event_key=EVENT_KEY,
    payload={
        "should_skip": False,
    },
    scope="foobarbaz",
)
