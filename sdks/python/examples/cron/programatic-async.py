from pydantic import BaseModel

from hatchet_sdk import Hatchet

hatchet = Hatchet()


class DynamicCronInput(BaseModel):
    name: str


async def create_cron() -> None:
    dynamic_cron_workflow = hatchet.workflow(
        name="CronWorkflow", input_validator=DynamicCronInput
    )

    # ❓ Create
    cron_trigger = await dynamic_cron_workflow.aio_create_cron(
        cron_name="customer-a-daily-report",
        expression="0 12 * * *",
        input=DynamicCronInput(name="John Doe"),
        additional_metadata={
            "customer_id": "customer-a",
        },
    )

    cron_trigger.metadata.id  # the id of the cron trigger
    # !!

    # ❓ List
    await hatchet.cron.aio_list()
    # !!

    # ❓ Get
    cron_trigger = await hatchet.cron.aio_get(cron_trigger=cron_trigger.metadata.id)
    # !!

    # ❓ Delete
    await hatchet.cron.aio_delete(cron_trigger=cron_trigger.metadata.id)
    # !!
