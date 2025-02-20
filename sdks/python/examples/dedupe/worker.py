import asyncio
from typing import Any

from hatchet_sdk import BaseWorkflow, ChildTriggerWorkflowOptions, Context, Hatchet
from hatchet_sdk.clients.admin import DedupeViolationErr

hatchet = Hatchet(debug=True)

dedupe_parent_wf = hatchet.declare_workflow(on_events=["parent:create"])


class DedupeParent(BaseWorkflow):
    config = dedupe_parent_wf.config

    @hatchet.step(timeout="1m")
    async def spawn(self, context: Context) -> dict[str, list[Any]]:
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
                    ).result()
                )
            except DedupeViolationErr as e:
                print(f"dedupe violation {e}")
                continue

        result = await asyncio.gather(*results)
        print(f"results {result}")

        return {"results": result}


dedupe_child_wf = hatchet.declare_workflow(on_events=["child:create"])


class DedupeChild(BaseWorkflow):
    config = dedupe_child_wf.config

    @hatchet.step()
    async def process(self, context: Context) -> dict[str, str]:
        await asyncio.sleep(3)

        print("child process")
        return {"status": "success"}

    @hatchet.step()
    async def process2(self, context: Context) -> dict[str, str]:
        print("child process2")
        return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker("fanout-worker", max_runs=100)
    worker.register_workflow(DedupeParent())
    worker.register_workflow(DedupeChild())
    worker.start()


if __name__ == "__main__":
    main()
