import asyncio
import os
import time
from typing import Any

from dotenv import load_dotenv

from hatchet_sdk import Context, sync_to_async
from hatchet_sdk.v2.hatchet import Hatchet

os.environ["PYTHONASYNCIODEBUG"] = "1"
load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.function()
async def fanout_sync_async(context: Context) -> dict[str, Any]:
    print("spawning child")

    context.put_stream("spawning...")
    results = []

    n = context.workflow_input().get("n", 10)

    start_time = time.time()
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

    execution_time = time.time() - start_time
    print(f"Completed in {execution_time:.2f} seconds")

    return {"results": result}


@hatchet.workflow(on_events=["child:create"])
class Child:
    ###### Example Functions ######
    def sync_blocking_function(self) -> dict[str, str]:
        time.sleep(5)
        return {"type": "sync_blocking"}

    @sync_to_async  # this makes the function async safe!
    def decorated_sync_blocking_function(self) -> dict[str, str]:
        time.sleep(5)
        return {"type": "decorated_sync_blocking"}

    @sync_to_async  # this makes the async function loop safe!
    async def async_blocking_function(self) -> dict[str, str]:
        time.sleep(5)
        return {"type": "async_blocking"}

    ###### Hatchet Steps ######
    @hatchet.step()
    async def handle_blocking_sync_in_async(self, context: Context) -> dict[str, str]:
        wrapped_blocking_function = sync_to_async(self.sync_blocking_function)

        # This will now be async safe!
        data = await wrapped_blocking_function()
        return {"blocking_status": "success", "data": data}

    @hatchet.step()
    async def handle_decorated_blocking_sync_in_async(
        self, context: Context
    ) -> dict[str, str]:
        data = await self.decorated_sync_blocking_function()
        return {"blocking_status": "success", "data": data}

    @hatchet.step()
    async def handle_blocking_async_in_async(self, context: Context) -> dict[str, str]:
        data = await self.async_blocking_function()
        return {"blocking_status": "success", "data": data}

    @hatchet.step()
    async def non_blocking_async(self, context: Context) -> dict[str, str]:
        await asyncio.sleep(5)
        return {"nonblocking_status": "success"}


def main() -> None:
    worker = hatchet.worker("fanout-worker", max_runs=50)
    worker.register_workflow(Child())
    worker.start()


if __name__ == "__main__":
    main()
