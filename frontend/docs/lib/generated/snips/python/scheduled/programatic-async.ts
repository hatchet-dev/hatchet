import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from datetime import datetime, timedelta\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\n\nasync def create_scheduled() -> None:\n    # > Create\n    scheduled_run = await hatchet.scheduled.aio_create(\n        workflow_name=\"simple-workflow\",\n        trigger_at=datetime.now() + timedelta(seconds=10),\n        input={\n            \"data\": \"simple-workflow-data\",\n        },\n        additional_metadata={\n            \"customer_id\": \"customer-a\",\n        },\n    )\n\n    scheduled_run.metadata.id  # the id of the scheduled run trigger\n\n    # > Delete\n    await hatchet.scheduled.aio_delete(scheduled_id=scheduled_run.metadata.id)\n\n    # > List\n    await hatchet.scheduled.aio_list()\n\n    # > Get\n    scheduled_run = await hatchet.scheduled.aio_get(\n        scheduled_id=scheduled_run.metadata.id\n    )\n",
  "source": "out/python/scheduled/programatic-async.py",
  "blocks": {
    "create": {
      "start": 10,
      "stop": 21
    },
    "delete": {
      "start": 24,
      "stop": 24
    },
    "list": {
      "start": 27,
      "stop": 27
    },
    "get": {
      "start": 30,
      "stop": 32
    }
  },
  "highlights": {}
};

export default snippet;
