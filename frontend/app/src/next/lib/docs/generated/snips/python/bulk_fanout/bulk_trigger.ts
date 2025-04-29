import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import asyncio\n\nfrom examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nhatchet = Hatchet()\n\n\nasync def main() -> None:\n    results = bulk_parent_wf.run_many(\n        workflows=[\n            bulk_parent_wf.create_bulk_run_item(\n                input=ParentInput(n=i),\n                options=TriggerWorkflowOptions(\n                    additional_metadata={\n                        'bulk-trigger': i,\n                        'hello-{i}': 'earth-{i}',\n                    }\n                ),\n            )\n            for i in range(20)\n        ],\n    )\n\n    for result in results:\n        print(result)\n\n\nif __name__ == '__main__':\n    asyncio.run(main())\n",
  source: 'out/python/bulk_fanout/bulk_trigger.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
