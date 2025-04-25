from datetime import timedelta
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


bulk_parent_wf = hatchet.workflow(name="BulkFanoutParent", input_validator=ParentInput)
bulk_child_wf = hatchet.workflow(name="BulkFanoutChild", input_validator=ChildInput)


# ❓ BulkFanoutParent
@bulk_parent_wf.task(execution_timeout=timedelta(minutes=5))
async def spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:
    # 👀 Create each workflow run to spawn
    child_workflow_runs = [
        bulk_child_wf.create_bulk_run_item(
            input=ChildInput(a=str(i)),
            key=f"child{i}",
            options=TriggerWorkflowOptions(additional_metadata={"hello": "earth"}),
        )
        for i in range(input.n)
    ]

    # 👀 Run workflows in bulk to improve performance
    spawn_results = await bulk_child_wf.aio_run_many(child_workflow_runs)

    return {"results": spawn_results}


# ‼️


@bulk_child_wf.task()
def process(input: ChildInput, ctx: Context) -> dict[str, str]:
    print(f"child process {input.a}")
    return {"status": "success " + input.a}


@bulk_child_wf.task()
def process2(input: ChildInput, ctx: Context) -> dict[str, str]:
    print("child process2")
    return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker(
        "fanout-worker", slots=40, workflows=[bulk_parent_wf, bulk_child_wf]
    )
    worker.start()


if __name__ == "__main__":
    main()
