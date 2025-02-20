import asyncio
from typing import cast

from dotenv import load_dotenv

from hatchet_sdk import Hatchet, WorkflowList

load_dotenv()

hatchet = Hatchet(debug=True)


async def main() -> None:
    workflow_list = await hatchet.rest.aio.workflow_list()
    rows = workflow_list.rows or []

    for workflow in rows:
        print(workflow.name)
        print(workflow.metadata.id)
        print(workflow.metadata.created_at)
        print(workflow.metadata.updated_at)


if __name__ == "__main__":
    asyncio.run(main())
