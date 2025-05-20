import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from hatchet_sdk import Context, EmptyModel, Hatchet\nfrom hatchet_sdk.exceptions import NonRetryableException\n\nhatchet = Hatchet(debug=True)\n\nnon_retryable_workflow = hatchet.workflow(name=\"NonRetryableWorkflow\")\n\n\n# > Non-retryable task\n@non_retryable_workflow.task(retries=1)\ndef should_not_retry(input: EmptyModel, ctx: Context) -> None:\n    raise NonRetryableException(\"This task should not retry\")\n\n\n\n\n@non_retryable_workflow.task(retries=1)\ndef should_retry_wrong_exception_type(input: EmptyModel, ctx: Context) -> None:\n    raise TypeError(\"This task should retry because it's not a NonRetryableException\")\n\n\n@non_retryable_workflow.task(retries=1)\ndef should_not_retry_successful_task(input: EmptyModel, ctx: Context) -> None:\n    pass\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"non-retry-worker\", workflows=[non_retryable_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/non_retryable/worker.py",
  "blocks": {
    "non_retryable_task": {
      "start": 10,
      "stop": 14
    }
  },
  "highlights": {}
};

export default snippet;
