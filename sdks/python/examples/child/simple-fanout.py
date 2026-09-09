from typing import Any

from examples.child.worker import SimpleInput, child_task
from hatchet_sdk.context.context import Context
from hatchet_sdk.hatchet import Hatchet

hatchet = Hatchet()


# > Running a Task from within a Task
@hatchet.task(name="SpawnTask")
async def spawn(input: None, ctx: Context) -> dict[str, Any]:
    # Simply run the task with the input we received
    result = await child_task.aio_run(
        input=SimpleInput(message="Hello, World!"),
    )

    return {"results": result}


# !!
