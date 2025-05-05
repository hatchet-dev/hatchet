import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "# > Simple\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\nfrom pydantic import BaseModel\n\nclass WorkflowInput(BaseModel):\n    group: str\n\nhatchet = Hatchet(debug=True)\n\n\nworkflow = hatchet.workflow(name='SimpleWorkflow', on_events=['test:event'], input_validator=WorkflowInput, event_filter_expression='input.group == 'shouldSkip'')\n\n@workflow.task()\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    print('executed step1')\n\n\ndef main() -> None:\n    worker = hatchet.worker('test-worker', slots=1, workflows=[workflow])\n    worker.start()\n\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/simple/worker.py',
  blocks: {
    simple: {
      start: 2,
      stop: 23,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
