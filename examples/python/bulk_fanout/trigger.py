from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf

bulk_parent_wf.run(
    ParentInput(n=999),
    additional_metadata={"no-dedupe": "world"},
)
