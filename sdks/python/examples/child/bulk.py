import asyncio

# ❓ Running a Task
from examples.child.worker import SimpleInput, child_task

child_task.run(SimpleInput(message="Hello, World!"))
# !!


async def main() -> None:
    # ❓ Bulk Run a Task
    greetings = ["Hello, World!", "Hello, Moon!", "Hello, Mars!"]

    results = await child_task.aio_run_many(
        [
            # run each greeting as a task in parallel
            child_task.create_bulk_run_item(
                input=SimpleInput(message=greeting),
            )
            for greeting in greetings
        ]
    )

    # this will await all results and return a list of results
    print(results)
    # !!

    # ❓ Running Multiple Tasks
    result1 = child_task.aio_run(SimpleInput(message="Hello, World!"))
    result2 = child_task.aio_run(SimpleInput(message="Hello, Moon!"))

    #  gather the results of the two tasks
    gather_results = await asyncio.gather(result1, result2)

    #  print the results of the two tasks
    print(gather_results[0]["transformed_message"])
    print(gather_results[1]["transformed_message"])
    # !!
