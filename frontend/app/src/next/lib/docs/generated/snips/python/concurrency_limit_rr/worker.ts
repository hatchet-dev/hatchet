import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import time\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n\n# > Concurrency Strategy With Key\nclass WorkflowInput(BaseModel):\n    group: str\n\n\nconcurrency_limit_rr_workflow = hatchet.workflow(\n    name=\'ConcurrencyDemoWorkflowRR\',\n    concurrency=ConcurrencyExpression(\n        expression=\'input.group\',\n        max_runs=1,\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n    input_validator=WorkflowInput,\n)\n\n\n@concurrency_limit_rr_workflow.task()\ndef step1(input: WorkflowInput, ctx: Context) -> None:\n    print(\'starting step1\')\n    time.sleep(2)\n    print(\'finished step1\')\n    pass\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'concurrency-demo-worker-rr\',\n        slots=10,\n        workflows=[concurrency_limit_rr_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/concurrency_limit_rr/worker.py',
  'blocks': {
    'concurrency_strategy_with_key': {
      'start': 16,
      'stop': 28
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
