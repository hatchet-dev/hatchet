import asyncio
from typing import TypedDict

from dotenv import load_dotenv

from hatchet_sdk import Context
from hatchet_sdk.v2.hatchet import Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


class MyResultType(TypedDict):
    return_string: str


@hatchet.function(default_priority=2)
async def high_prio_func(context: Context) -> MyResultType:
    await asyncio.sleep(5)
    return MyResultType(return_string="High Priority Return")


@hatchet.function(default_priority=1)
async def low_prio_func(context: Context) -> MyResultType:
    await asyncio.sleep(5)
    return MyResultType(return_string="Low Priority Return")


def main() -> None:
    worker = hatchet.worker("example-priority-worker", max_runs=1)
    hatchet.admin.run(high_prio_func, {"test": "test"})
    hatchet.admin.run(high_prio_func, {"test": "test"})
    hatchet.admin.run(low_prio_func, {"test": "test"})
    hatchet.admin.run(low_prio_func, {"test": "test"})
    worker.start()


if __name__ == "__main__":
    main()
