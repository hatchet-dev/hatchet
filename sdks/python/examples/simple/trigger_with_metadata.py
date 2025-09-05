from hatchet_sdk import TriggerWorkflowOptions
from examples.simple.worker import simple

# > Trigger with metadata
simple.run(
    options=TriggerWorkflowOptions(
        additional_metadata={
            "source": "api"  # Arbitrary key-value pair
        }
    )
)
# !!
