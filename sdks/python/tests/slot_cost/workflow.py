import asyncio

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

SLOT_COST = 5
WORKER_SLOTS = 2 * SLOT_COST - 1

# Long enough that two runs overlap if the worker can hold both at once, which is the failure the
# e2e test rules out.
SLEEP_TIME = 2

slot_cost_workflow = hatchet.workflow(name="slot-cost-e2e")


@slot_cost_workflow.task(slot_cost=SLOT_COST)
async def heavy_task(input: EmptyModel, ctx: Context) -> None:
    await asyncio.sleep(SLEEP_TIME)
