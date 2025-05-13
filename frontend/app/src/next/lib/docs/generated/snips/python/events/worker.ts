import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\n\nclass EventWorkflowInput(BaseModel):\n    should_skip: bool\n\n\nhatchet = Hatchet()\n\n# > Event trigger\nevent_workflow = hatchet.workflow(\n    name='EventWorkflow',\n    on_events=['user:create'],\n    input_validator=EventWorkflowInput,\n)\n\n\n@event_workflow.task()\ndef task(input: EventWorkflowInput, ctx: Context) -> None:\n    print('event received')\n\n\ndef main() -> None:\n    worker = hatchet.worker(name='EventWorker', workflows=[event_workflow])\n\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/events/worker.py',
  blocks: {
    event_trigger: {
      start: 13,
      stop: 17,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
