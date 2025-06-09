import asyncio
from datetime import timedelta
from typing import Any

from hatchet_sdk import Context, EmptyModel, Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.admin import DedupeViolationError

hatchet = Hatchet(debug=True)

dedupe_parent_wf = hatchet.workflow(name="DedupeParent")
dedupe_child_wf = hatchet.workflow(name="DedupeChild")


@dedupe_parent_wf.task(execution_timeout=timedelta(minutes=1))
async def spawn(input: EmptyModel, ctx: Context) -> dict[str, list[Any]]:
    print("spawning child")

    results = []

    for i in range(2):
        try:
            results.append(
                (
                    dedupe_child_wf.aio_run(
                        options=TriggerWorkflowOptions(
                            additional_metadata={"dedupe": "test"}, key=f"child{i}"
                        ),
                    )
                )
            )
        except DedupeViolationError as e:
            print(f"dedupe violation {e}")
            continue

    result = await asyncio.gather(*results)
    print(f"results {result}")

    return {"results": result}


@dedupe_child_wf.task()
async def process(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(3)

    print("child process")
    return {"status": "success"}


@dedupe_child_wf.task()
async def process2(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("child process2")
    return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker(
        "fanout-worker", slots=100, workflows=[dedupe_parent_wf, dedupe_child_wf]
    )
    worker.start()


if __name__ == "__main__":
    main()
