import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# > Event trigger\nevent_workflow = hatchet.workflow(name='EventWorkflow', on_events=['user:create'])\n\n\n@event_workflow.task()\ndef task(input: EmptyModel, ctx: Context) -> None:\n    print('event received')\n\ndef main() -> None:\n    worker = hatchet.worker(name='EventWorker', workflows=[event_workflow])\n\n    worker.start()\n\nif __name__ == '__main__':\n    main()",
  source: 'out/python/events/worker.py',
  blocks: {
    event_trigger: {
      start: 6,
      stop: 6,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
