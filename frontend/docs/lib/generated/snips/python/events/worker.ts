import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, DefaultFilter, Hatchet\n\nhatchet = Hatchet()\n\n\n# > Event trigger\nEVENT_KEY = \"user:create\"\nSECONDARY_KEY = \"foobarbaz\"\nWILDCARD_KEY = \"subscription:*\"\n\n\nclass EventWorkflowInput(BaseModel):\n    should_skip: bool\n\n\nevent_workflow = hatchet.workflow(\n    name=\"EventWorkflow\",\n    on_events=[EVENT_KEY, SECONDARY_KEY, WILDCARD_KEY],\n    input_validator=EventWorkflowInput,\n)\n\n# > Event trigger with filter\nevent_workflow_with_filter = hatchet.workflow(\n    name=\"EventWorkflow\",\n    on_events=[EVENT_KEY, SECONDARY_KEY, WILDCARD_KEY],\n    input_validator=EventWorkflowInput,\n    default_filters=[\n        DefaultFilter(\n            expression=\"true\",\n            scope=\"example-scope\",\n            payload={\n                \"main_character\": \"Anna\",\n                \"supporting_character\": \"Stiva\",\n                \"location\": \"Moscow\",\n            },\n        )\n    ],\n)\n\n\n@event_workflow.task()\ndef task(input: EventWorkflowInput, ctx: Context) -> dict[str, str]:\n    print(\"event received\")\n\n    return dict(ctx.filter_payload)\n\n\ndef main() -> None:\n    worker = hatchet.worker(name=\"EventWorker\", workflows=[event_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/events/worker.py",
  "blocks": {
    "event_trigger": {
      "start": 9,
      "stop": 22
    },
    "event_trigger_with_filter": {
      "start": 25,
      "stop": 40
    }
  },
  "highlights": {}
};

export default snippet;
