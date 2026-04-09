from examples.runtime_affinity.worker import runtime_affinity_workflow
from hatchet_sdk import DesiredWorkerLabel

res = runtime_affinity_workflow.run_many(
    [
        runtime_affinity_workflow.create_bulk_run_item(
            desired_worker_labels=[
                DesiredWorkerLabel(
                    key="affinity",
                    value="foo",
                    required=True,
                ),
            ],
        )
        for _ in range(5)
    ]
)

ids = set()
for run in res:
    for id in [x["worker_id"] for x in list(run.values())]:
        ids.add(id)

print(ids)
