from examples.simple.worker import hatchet
from hatchet_sdk.clients.admin import CreateFilterRequest
from hatchet_sdk.clients.events import PushEventOptions

hatchet._client.admin.put_filter(
    CreateFilterRequest(
        workflow_id="1563ea96-3d4a-4b62-ae68-94bb11ea3bb1",
        expression="input.shouldSkipThis == true",
        resource_hint="foobar",
        payload={"test": "test"},
    )
)

hatchet.event.push(
    "workflow-filters:test:1",
    {"shouldSkipThis": True},
    PushEventOptions(
        resource_hint="foobar",
        additional_metadata={
            "shouldSkipThis": True,
        },
    ),
)

hatchet.event.push(
    "workflow-filters:test:1",
    {"shouldSkipThis": False},
    PushEventOptions(
        resource_hint="foobar",
        additional_metadata={
            "shouldSkipThis": False,
        },
    ),
)
