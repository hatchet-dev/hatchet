import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "# > Simple\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n@hatchet.task(\n    name='SimpleWorkflowEventFiltering', on_events=['workflow-filters:test:1']\n)\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    print('executed step1')\n\n\ndef main() -> None:\n    worker = hatchet.worker('test-worker', slots=1, workflows=[step1])\n    worker.start()\n\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/simple/worker.py',
  blocks: {
    simple: {
      start: 2,
      stop: 19,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
