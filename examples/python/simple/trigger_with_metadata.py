from examples.simple.worker import simple
from hatchet_sdk import RunWorkflowOptions

# > Trigger with metadata
simple.run(
    options=RunWorkflowOptions(
        additional_metadata={"source": "api"}  # Arbitrary key-value pair
    )
)
