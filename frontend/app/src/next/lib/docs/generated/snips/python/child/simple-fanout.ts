import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from typing import Any\n\nfrom examples.child.worker import SimpleInput, child_task\nfrom hatchet_sdk.context.context import Context\nfrom hatchet_sdk.hatchet import Hatchet\nfrom hatchet_sdk.runnables.types import EmptyModel\n\nhatchet = Hatchet(debug=True)\n\n\n# > Running a Task from within a Task\n@hatchet.task(name=\'SpawnTask\')\nasync def spawn(input: EmptyModel, ctx: Context) -> dict[str, Any]:\n    # Simply run the task with the input we received\n    result = await child_task.aio_run(\n        input=SimpleInput(message=\'Hello, World!\'),\n    )\n\n    return {\'results\': result}\n\n\n',
  'source': 'out/python/child/simple-fanout.py',
  'blocks': {
    'running_a_task_from_within_a_task': {
      'start': 12,
      'stop': 21
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
