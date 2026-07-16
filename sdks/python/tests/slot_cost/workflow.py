import asyncio

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()

SLOT_COST = 5
WORKER_SLOTS = 2 * SLOT_COST - 1

# Long enough that two runs overlap if the worker can hold both at once, which is the failure the
# e2e test rules out.
SLEEP_TIME = 2


@hatchet.task(slot_cost=SLOT_COST)
async def slot_cost_test_heavy_task(input: None, ctx: Context) -> None:
    await asyncio.sleep(SLEEP_TIME)
