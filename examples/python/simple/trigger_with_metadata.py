from examples.simple.worker import simple
from hatchet_sdk import TriggerWorkflowOptions

# > Trigger with metadata
simple.run(
    options=TriggerWorkflowOptions(
        additional_metadata={"source": "api"}  # Arbitrary key-value pair
    )
)
