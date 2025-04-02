import asyncio
from typing import List

# ❓ Running a Task
from examples.simple.worker import SimpleInput, step1

step1.run(SimpleInput(message="Hello, World!"))
# !!


async def main() -> None:
    # ❓ Bulk Run a Task
    greetings = ["Hello, World!", "Hello, Moon!", "Hello, Mars!"]

    results = await step1.aio_run_many(
        [
            # run each greeting as a task in parallel
            step1.create_bulk_run_item(
                input=SimpleInput(message=greeting),
            )
            for greeting in greetings
        ]
    )

    # this will await all results and return a list of results
    print(results)
    # !!

    # ❓ Running Multiple Tasks
    result1 = step1.aio_run(SimpleInput(message="Hello, World!"))
    result2 = step1.aio_run(SimpleInput(message="Hello, Moon!"))

    #  gather the results of the two tasks
    results = list(await asyncio.gather(result1, result2))

    #  print the results of the two tasks
    print(results[0].transformed_message)
    print(results[1].transformed_message)
    # !!
