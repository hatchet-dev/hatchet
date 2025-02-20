import asyncio
from typing import Any

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.clients.admin import ChildWorkflowRunDict

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["parent:create"])
class BulkParent:
    @hatchet.step(timeout="5m")
    async def spawn(self, context: Context) -> dict[str, list[Any]]:
        print("spawning child")

        context.put_stream("spawning...")
        results = []

        n = context.workflow_input().get("n", 100)

        child_workflow_runs: list[ChildWorkflowRunDict] = []

        for i in range(n):

            child_workflow_runs.append(
                {
                    "workflow_name": "BulkChild",
                    "input": {"a": str(i)},
                    "key": f"child{i}",
                    "options": {"additional_metadata": {"hello": "earth"}},
                }
            )

        if len(child_workflow_runs) == 0:
            return {}

        spawn_results = await context.aio.spawn_workflows(child_workflow_runs)

        results = await asyncio.gather(
            *[workflowRunRef.result() for workflowRunRef in spawn_results],
            return_exceptions=True,
        )

        print("finished spawning children")

        for result in results:
            if isinstance(result, Exception):
                print(f"An error occurred: {result}")
            else:
                print(result)

        return {"results": results}


@hatchet.workflow(on_events=["child:create"])
class BulkChild:
    @hatchet.step()
    def process(self, context: Context) -> dict[str, str]:
        a = context.workflow_input()["a"]
        print(f"child process {a}")
        context.put_stream("child 1...")
        return {"status": "success " + a}

    @hatchet.step()
    def process2(self, context: Context) -> dict[str, str]:
        print("child process2")
        context.put_stream("child 2...")
        return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker("fanout-worker", max_runs=40)
    worker.register_workflow(BulkParent())
    worker.register_workflow(BulkChild())
    worker.start()


if __name__ == "__main__":
    main()
