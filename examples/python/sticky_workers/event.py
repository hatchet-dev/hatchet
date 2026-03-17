from examples.sticky_workers.worker import sticky_workflow

sticky_workflow.run(
    additional_metadata={"hello": "moon"},
)
