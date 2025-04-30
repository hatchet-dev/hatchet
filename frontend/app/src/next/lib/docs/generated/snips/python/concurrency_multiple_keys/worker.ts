import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import asyncio\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\nSLEEP_TIME = 2\nDIGIT_MAX_RUNS = 8\nNAME_MAX_RUNS = 3\n\n\n# > Concurrency Strategy With Key\nclass WorkflowInput(BaseModel):\n    name: str\n    digit: str\n\n\nconcurrency_multiple_keys_workflow = hatchet.workflow(\n    name=\'ConcurrencyWorkflowManyKeys\',\n    input_validator=WorkflowInput,\n)\n\n\n@concurrency_multiple_keys_workflow.task(\n    concurrency=[\n        ConcurrencyExpression(\n            expression=\'input.digit\',\n            max_runs=DIGIT_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n        ConcurrencyExpression(\n            expression=\'input.name\',\n            max_runs=NAME_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n    ]\n)\nasync def concurrency_task(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'concurrency-worker-multiple-keys\',\n        slots=10,\n        workflows=[concurrency_multiple_keys_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/concurrency_multiple_keys/worker.py',
  'blocks': {
    'concurrency_strategy_with_key': {
      'start': 20,
      'stop': 28
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
