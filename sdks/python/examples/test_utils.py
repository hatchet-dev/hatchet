import asyncio

from hatchet_sdk import Hatchet, RunStatus


async def wait_for_running_status(
    hatchet: Hatchet, run_id: str, timeout: float = 60.0
) -> None:
    """Poll until the workflow run reaches RUNNING status or timeout is exceeded."""
    interval = 0.5
    max_iters = int(timeout / interval)
    for _ in range(max_iters):
        run = await hatchet.runs.aio_get_details(run_id)
        if run.status == RunStatus.RUNNING:
            return
        await asyncio.sleep(interval)
