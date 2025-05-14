from examples.simple.worker import EVENT_KEY, hatchet, step1
from hatchet_sdk.clients.events import PushEventOptions

SCOPE = "foobar"
hatchet.filters.create(
    workflow_id=step1.id,
    expression="input.shouldSkipThis == false",
    scope=SCOPE,
    payload={"test": "test"},
)

hatchet.event.push(
    EVENT_KEY,
    {"shouldSkipThis": True},
    PushEventOptions(
        ## If no scope is provided, all events pushed will trigger workflows
        ## (i.e. the filter will not apply)
        scope=SCOPE,
        additional_metadata={
            "shouldSkipThis": True,
        },
    ),
)

hatchet.event.push(
    EVENT_KEY,
    {"shouldSkipThis": False},
    PushEventOptions(
        ## If no scope is provided, all events pushed will trigger workflows
        ## (i.e. the filter will not apply)
        scope=SCOPE,
        additional_metadata={
            "shouldSkipThis": False,
        },
    ),
)
