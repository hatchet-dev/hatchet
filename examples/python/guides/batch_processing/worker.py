from typing import Any

from hatchet_sdk import Context, Hatchet
from pydantic import BaseModel

hatchet = Hatchet(debug=True)


# > Step 01 Define Parent Task
class BatchInput(BaseModel):
    items: list[str]


class ItemInput(BaseModel):
    item_id: str


parent_wf = hatchet.workflow(name="BatchParent", input_validator=BatchInput)
child_wf = hatchet.workflow(name="BatchChild", input_validator=ItemInput)


@parent_wf.task()
async def spawn_children(input: BatchInput, ctx: Context) -> dict[str, Any]:
    """Parent fans out to one child per item."""
    results = []
    for item_id in input.items:
        result = await child_wf.aio_run(input=ItemInput(item_id=item_id))
        results.append(result)
    return {"processed": len(results), "results": results}




# > Step 02 Fan Out Children
async def _fan_out(input: BatchInput) -> list:
    results = []
    for item_id in input.items:
        result = await child_wf.aio_run(input=ItemInput(item_id=item_id))
        results.append(result)
    return results
# Hatchet distributes child runs across available workers.


# > Step 03 Process Item
@child_wf.task()
async def process_item(input: ItemInput, ctx: Context) -> dict[str, str]:
    """Child processes a single item."""
    return {"status": "done", "item_id": input.item_id}




def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "batch-worker",
        slots=20,
        workflows=[parent_wf, child_wf],
    )
    worker.start()


if __name__ == "__main__":
    main()
