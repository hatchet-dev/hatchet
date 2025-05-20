import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.rate_limit import RateLimit, RateLimitDuration\n\nhatchet = Hatchet(debug=True)\n\n\n# > Workflow\nclass RateLimitInput(BaseModel):\n    user_id: str\n\n\nrate_limit_workflow = hatchet.workflow(\n    name=\"RateLimitWorkflow\", input_validator=RateLimitInput\n)\n\n\n\n# > Static\nRATE_LIMIT_KEY = \"test-limit\"\n\n\n@rate_limit_workflow.task(rate_limits=[RateLimit(static_key=RATE_LIMIT_KEY, units=1)])\ndef step_1(input: RateLimitInput, ctx: Context) -> None:\n    print(\"executed step_1\")\n\n\n\n# > Dynamic\n\n\n@rate_limit_workflow.task(\n    rate_limits=[\n        RateLimit(\n            dynamic_key=\"input.user_id\",\n            units=1,\n            limit=10,\n            duration=RateLimitDuration.MINUTE,\n        )\n    ]\n)\ndef step_2(input: RateLimitInput, ctx: Context) -> None:\n    print(\"executed step_2\")\n\n\n\n\ndef main() -> None:\n    hatchet.rate_limits.put(RATE_LIMIT_KEY, 2, RateLimitDuration.SECOND)\n\n    worker = hatchet.worker(\n        \"rate-limit-worker\", slots=10, workflows=[rate_limit_workflow]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/rate_limit/worker.py",
  "blocks": {
    "workflow": {
      "start": 10,
      "stop": 17
    },
    "static": {
      "start": 21,
      "stop": 28
    },
    "dynamic": {
      "start": 31,
      "stop": 46
    }
  },
  "highlights": {}
};

export default snippet;
