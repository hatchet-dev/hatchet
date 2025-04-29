import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import time\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n\n# > Workflow\nclass WorkflowInput(BaseModel):\n    run: int\n    group_key: str\n\n\nconcurrency_limit_workflow = hatchet.workflow(\n    name=\'ConcurrencyDemoWorkflow\',\n    concurrency=ConcurrencyExpression(\n        expression=\'input.group_key\',\n        max_runs=5,\n        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,\n    ),\n    input_validator=WorkflowInput,\n)\n\n\n\n@concurrency_limit_workflow.task()\ndef step1(input: WorkflowInput, ctx: Context) -> dict[str, Any]:\n    time.sleep(3)\n    print(\'executed step1\')\n    return {\'run\': input.run}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'concurrency-demo-worker\', slots=10, workflows=[concurrency_limit_workflow]\n    )\n\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/concurrency_limit/worker.py',
  'blocks': {
    'workflow': {
      'start': 17,
      'stop': 31
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
