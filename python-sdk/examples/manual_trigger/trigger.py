from hatchet_sdk import new_client
from dotenv import load_dotenv

load_dotenv()

client = new_client()

client.admin.run_workflow("ManualTriggerWorkflow", {
    "test": "test"
})
