from pydantic import BaseModel

from hatchet_sdk import Hatchet

hatchet = Hatchet()


class DynamicCronInput(BaseModel):
    name: str


dynamic_cron_workflow = hatchet.workflow(
    name="CronWorkflow", input_validator=DynamicCronInput
)

# ❓ Create
cron_trigger = dynamic_cron_workflow.create_cron(
    cron_name="customer-a-daily-report",
    expression="0 12 * * *",
    input=DynamicCronInput(name="John Doe"),
    additional_metadata={
        "customer_id": "customer-a",
    },
)


id = cron_trigger.metadata.id  # the id of the cron trigger
# !!

# ❓ List
cron_triggers = hatchet.cron.list()
# !!

# ❓ Get
cron_trigger = hatchet.cron.get(cron_trigger=cron_trigger.metadata.id)
# !!

# ❓ Delete
hatchet.cron.delete(cron_trigger=cron_trigger.metadata.id)
# !!
