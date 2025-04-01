from datetime import timedelta
from typing import Any

from examples.simple.worker import SimpleInput, step1
from hatchet_sdk.context.context import Context
from hatchet_sdk.hatchet import Hatchet
from hatchet_sdk.runnables.types import EmptyModel

hatchet = Hatchet(debug=True)


# ❓ Running a Task from within a Task
parent_wf = hatchet.task(name="parent_task")


@parent_wf.task(execution_timeout=timedelta(minutes=5))
async def spawn(input: EmptyModel, ctx: Context) -> dict[str, Any]:

    # Simply run the task with the input we received
    result = await step1.aio_run(
        input=SimpleInput(message="Hello, World!"),
    )

    return {"results": result}


# ‼️
