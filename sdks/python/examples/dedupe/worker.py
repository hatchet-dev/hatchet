import asyncio
from typing import Any

from hatchet_sdk import ChildTriggerWorkflowOptions, Context, EmptyModel, Hatchet
from hatchet_sdk.clients.admin import DedupeViolationErr

hatchet = Hatchet(debug=True)

dedupe_parent_wf = hatchet.workflow(name="DedupeParent", on_events=["parent:create"])


@dedupe_parent_wf.task(timeout="1m")
async def spawn(input: EmptyModel, context: Context) -> dict[str, list[Any]]:
    print("spawning child")

    results = []

    for i in range(2):
        try:
            results.append(
                (
                    await context.aio_spawn_workflow(
                        "DedupeChild",
                        {"a": str(i)},
                        key=f"child{i}",
                        options=ChildTriggerWorkflowOptions(
                            additional_metadata={"dedupe": "test"}
                        ),
                    )
                ).aio_result()
            )
        except DedupeViolationErr as e:
            print(f"dedupe violation {e}")
            continue

    result = await asyncio.gather(*results)
    print(f"results {result}")

    return {"results": result}


dedupe_child_wf = hatchet.workflow(name="DedupeChild", on_events=["child:create"])


@dedupe_child_wf.task()
async def process(input: EmptyModel, context: Context) -> dict[str, str]:
    await asyncio.sleep(3)

    print("child process")
    return {"status": "success"}


@dedupe_child_wf.task()
async def process2(input: EmptyModel, context: Context) -> dict[str, str]:
    print("child process2")
    return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker(
        "fanout-worker", max_runs=100, workflows=[dedupe_parent_wf, dedupe_child_wf]
    )
    worker.start()


if __name__ == "__main__":
    main()
