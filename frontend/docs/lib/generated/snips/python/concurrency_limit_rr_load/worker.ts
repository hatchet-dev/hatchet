import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import random\nimport time\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n\nclass LoadRRInput(BaseModel):\n    group: str\n\n\nload_rr_workflow = hatchet.workflow(\n    name=\"LoadRoundRobin\",\n    on_events=[\"concurrency-test\"],\n    concurrency=ConcurrencyExpression(\n        expression=\"input.group\",\n        max_runs=1,\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n    input_validator=LoadRRInput,\n)\n\n\n@load_rr_workflow.on_failure_task()\ndef on_failure(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"on_failure\")\n    return {\"on_failure\": \"on_failure\"}\n\n\n@load_rr_workflow.task()\ndef step1(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"starting step1\")\n    time.sleep(random.randint(2, 20))\n    print(\"finished step1\")\n    return {\"step1\": \"step1\"}\n\n\n@load_rr_workflow.task(\n    retries=3,\n    backoff_factor=5,\n    backoff_max_seconds=60,\n)\ndef step2(sinput: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"starting step2\")\n    if random.random() < 0.5:  # 1% chance of failure\n        raise Exception(\"Random failure in step2\")\n    time.sleep(2)\n    print(\"finished step2\")\n    return {\"step2\": \"step2\"}\n\n\n@load_rr_workflow.task()\ndef step3(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"starting step3\")\n    time.sleep(0.2)\n    print(\"finished step3\")\n    return {\"step3\": \"step3\"}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"concurrency-demo-worker-rr\", slots=50, workflows=[load_rr_workflow]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/concurrency_limit_rr_load/worker.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
