from dotenv import load_dotenv

from hatchet_sdk import Hatchet

load_dotenv()

hatchet = Hatchet()


async def create_cron() -> None:
    # ❓ Create
    cron_trigger = await hatchet.cron.aio.create(
        workflow_name="simple-cron-workflow",
        cron_name="customer-a-daily-report",
        expression="0 12 * * *",
        input={
            "name": "John Doe",
        },
        additional_metadata={
            "customer_id": "customer-a",
        },
    )

    id = cron_trigger.metadata.id  # the id of the cron trigger
    # !!

    # ❓ List
    cron_triggers = await hatchet.cron.aio.list()
    # !!

    # ❓ Get
    cron_trigger = await hatchet.cron.aio.get(cron_trigger=cron_trigger.metadata.id)
    # !!

    # ❓ Delete
    await hatchet.cron.aio.delete(cron_trigger=cron_trigger.metadata.id)
    # !!
