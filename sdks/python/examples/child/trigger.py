# ruff: noqa: E402

import asyncio

# ❓ Running a Task
from examples.child.worker import SimpleInput, child_task

child_task.run(SimpleInput(message="Hello, World!"))
# !!

# ❓ Schedule a Task
from datetime import datetime, timedelta

child_task.schedule(
    datetime.now() + timedelta(minutes=5), SimpleInput(message="Hello, World!")
)
# !!


async def main() -> None:
    # ❓ Running a Task AIO
    result = await child_task.aio_run(SimpleInput(message="Hello, World!"))
    # !!

    print(result)

    # ❓ Running Multiple Tasks
    result1 = child_task.aio_run(SimpleInput(message="Hello, World!"))
    result2 = child_task.aio_run(SimpleInput(message="Hello, Moon!"))

    #  gather the results of the two tasks
    results = await asyncio.gather(result1, result2)

    #  print the results of the two tasks
    print(results[0]["transformed_message"])
    print(results[1]["transformed_message"])
    # !!
