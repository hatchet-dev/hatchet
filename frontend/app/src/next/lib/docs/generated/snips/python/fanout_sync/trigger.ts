import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import asyncio\n\nfrom examples.fanout_sync.worker import ParentInput, sync_fanout_parent\nfrom hatchet_sdk import Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet()\n\n\nasync def main() -> None:\n    sync_fanout_parent.run(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={\'hello\': \'moon\'}),\n    )\n\n\nif __name__ == \'__main__\':\n    asyncio.run(main())\n',
  'source': 'out/python/fanout_sync/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
