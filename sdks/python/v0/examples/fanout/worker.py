import asyncio
from typing import Any

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["parent:create"])
class Parent:
    @hatchet.step(timeout="5m")
    async def spawn(self, context: Context) -> dict[str, Any]:
        print("spawning child")

        context.put_stream("spawning...")
        results = []

        n = context.workflow_input().get("n", 100)

        for i in range(n):
            results.append(
                (
                    await context.aio.spawn_workflow(
                        "Child",
                        {"a": str(i)},
                        key=f"child{i}",
                        options={"additional_metadata": {"hello": "earth"}},
                    )
                ).result()
            )

        result = await asyncio.gather(*results)
        print(f"results {result}")

        return {"results": result}


@hatchet.workflow(on_events=["child:create"])
class Child:
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
    worker.register_workflow(Parent())
    worker.register_workflow(Child())
    worker.start()


if __name__ == "__main__":
    main()
