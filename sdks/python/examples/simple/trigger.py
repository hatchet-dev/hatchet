import asyncio
# ❓ Running a Task
from examples.simple.worker import SimpleInput, step1

step1.run(SimpleInput(message="Hello, World!"))
# !!

async def main():
    # ❓ Running a Task AIO
    result = await step1.aio_run(SimpleInput(message="Hello, World!"))
    # !!

    # ❓ Running Multiple Tasks
    result1 = step1.aio_run(SimpleInput(message="Hello, World!"))
    result2 = step1.aio_run(SimpleInput(message="Hello, Moon!"))

    #  gather the results of the two tasks
    results = await asyncio.gather(result1, result2)

    #  print the results of the two tasks
    print(results[0].TransformedMessage)
    print(results[1].TransformedMessage)
    # !!
