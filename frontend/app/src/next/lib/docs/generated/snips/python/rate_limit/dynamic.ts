import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.rate_limit import RateLimit\n\nhatchet = Hatchet(debug=True)\n\n\nclass DynamicRateLimitInput(BaseModel):\n    group: str\n    units: int\n    limit: int\n\n\ndynamic_rate_limit_workflow = hatchet.workflow(\n    name="DynamicRateLimitWorkflow", input_validator=DynamicRateLimitInput\n)\n\n\n@dynamic_rate_limit_workflow.task(\n    rate_limits=[\n        RateLimit(\n            dynamic_key=\'"LIMIT:"+input.group\',\n            units="input.units",\n            limit="input.limit",\n        )\n    ]\n)\ndef step1(input: DynamicRateLimitInput, ctx: Context) -> None:\n    print("executed step1")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "rate-limit-worker", slots=10, workflows=[dynamic_rate_limit_workflow]\n    )\n    worker.start()\n',
  source: 'out/python/rate_limit/dynamic.py',
  blocks: {},
  highlights: {},
};

export default snippet;
