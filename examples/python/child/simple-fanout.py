from typing import Any

from examples.child.worker import SimpleInput, child_task
from hatchet_sdk.context.context import Context
from hatchet_sdk.hatchet import Hatchet
from hatchet_sdk.runnables.types import EmptyModel

hatchet = Hatchet(debug=True)

# ❓ Running a Task from within a Task
@hatchet-dev/typescript-sdk.task(name="SpawnTask")
async def spawn(input: EmptyModel, ctx: Context) -> dict[str, Any]:
    # Simply run the task with the input we received
    result = await child_task.aio_run(
        input=SimpleInput(message="Hello, World!"),
    )

    return {"results": result}

# ‼️
