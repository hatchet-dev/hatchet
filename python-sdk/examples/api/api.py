from dotenv import load_dotenv

from hatchet_sdk import Hatchet, WorkflowList

load_dotenv()

hatchet = Hatchet(debug=True)

list: WorkflowList = hatchet.client.rest().workflow_list()

for workflow in list.rows:
    print(workflow.name)
    print(workflow.metadata.id)
    print(workflow.metadata.created_at)
    print(workflow.metadata.updated_at)
