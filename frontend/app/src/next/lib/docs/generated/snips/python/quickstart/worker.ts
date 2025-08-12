import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from .hatchet_client import hatchet\nfrom .workflows.first_task import first_task\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "first-worker",\n        slots=10,\n        workflows=[first_task],\n    )\n    worker.start()\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/quickstart/worker.py',
  blocks: {},
  highlights: {},
};

export default snippet;
