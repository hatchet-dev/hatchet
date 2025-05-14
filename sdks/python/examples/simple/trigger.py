from examples.simple.worker import hatchet
from hatchet_sdk.clients.events import PushEventOptions

hatchet.filters.create(
    workflow_id="1563ea96-3d4a-4b62-ae68-94bb11ea3bb1",
    expression="input.shouldSkipThis == true",
    scope="foobar",
    payload={"test": "test"},
)

hatchet.event.push(
    "workflow-filters:test:2",
    {"shouldSkipThis": True},
    PushEventOptions(
        scope="foobar",
        additional_metadata={
            "shouldSkipThis": True,
        },
    ),
)

hatchet.event.push(
    "workflow-filters:test:2",
    {"shouldSkipThis": False},
    PushEventOptions(
        scope="foobar",
        additional_metadata={
            "shouldSkipThis": False,
        },
    ),
)
