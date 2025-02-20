from dotenv import load_dotenv

from hatchet_sdk import Hatchet

load_dotenv()

hatchet = Hatchet()

# ❓ Create
cron_trigger = hatchet.cron.create(
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
cron_triggers = hatchet.cron.list()
# !!

# ❓ Get
cron_trigger = hatchet.cron.get(cron_trigger=cron_trigger.metadata.id)
# !!

# ❓ Delete
hatchet.cron.delete(cron_trigger=cron_trigger.metadata.id)
# !!
