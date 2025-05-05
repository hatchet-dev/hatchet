import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import asyncio\n\nfrom examples.fanout.worker import ParentInput, parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nhatchet = Hatchet()\n\n\nasync def main() -> None:\n    await parent_wf.aio_run(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={'hello': 'moon'}),\n    )\n\n\nif __name__ == '__main__':\n    asyncio.run(main())\n",
  source: 'out/python/fanout/trigger.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
