// Generated from /Users/gabrielruttner/dev/hatchet/examples/python/logger/worker.py
export const content = "from examples.logger.client import hatchet\nfrom examples.logger.workflow import logging_workflow\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"logger-worker\", slots=5, workflows=[logging_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n";
export const language = "py";
