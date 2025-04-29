import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import time\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    EmptyModel,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n# > Default priority\nDEFAULT_PRIORITY = 1\nSLEEP_TIME = 0.25\n\npriority_workflow = hatchet.workflow(\n    name=\'PriorityWorkflow\',\n    default_priority=DEFAULT_PRIORITY,\n    concurrency=ConcurrencyExpression(\n        max_runs=1,\n        expression=\'\'true\'\',\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n)\n\n\n@priority_workflow.task()\ndef priority_task(input: EmptyModel, ctx: Context) -> None:\n    print(\'Priority:\', ctx.priority)\n    time.sleep(SLEEP_TIME)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'priority-worker\',\n        slots=1,\n        workflows=[priority_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/priority/worker.py',
  'blocks': {
    'default_priority': {
      'start': 14,
      'stop': 25
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
