import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import asyncio\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\nSLEEP_TIME = 2\nDIGIT_MAX_RUNS = 8\nNAME_MAX_RUNS = 3\n\n\n# > Multiple Concurrency Keys\nclass WorkflowInput(BaseModel):\n    name: str\n    digit: str\n\n\nconcurrency_workflow_level_workflow = hatchet.workflow(\n    name='ConcurrencyWorkflowManyKeys',\n    input_validator=WorkflowInput,\n    concurrency=[\n        ConcurrencyExpression(\n            expression='input.digit',\n            max_runs=DIGIT_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n        ConcurrencyExpression(\n            expression='input.name',\n            max_runs=NAME_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n    ],\n)\n\n\n@concurrency_workflow_level_workflow.task()\nasync def task_1(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\n\n@concurrency_workflow_level_workflow.task()\nasync def task_2(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        'concurrency-worker-workflow-level',\n        slots=10,\n        workflows=[concurrency_workflow_level_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/concurrency_workflow_level/worker.py',
  blocks: {
    multiple_concurrency_keys: {
      start: 20,
      stop: 40,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
