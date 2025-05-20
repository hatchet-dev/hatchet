import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from examples.logger.client import hatchet\nfrom examples.logger.workflow import logging_workflow\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"logger-worker\", slots=5, workflows=[logging_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/logger/worker.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
