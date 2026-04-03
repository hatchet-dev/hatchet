from examples.simple.worker import simple

# > Trigger with metadata
simple.run(
    additional_metadata={"source": "api"},  # Arbitrary key-value pair
)
# !!
