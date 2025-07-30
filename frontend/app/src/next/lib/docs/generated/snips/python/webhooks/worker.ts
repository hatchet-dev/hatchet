import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    '# > Webhooks\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\nclass WebhookInput(BaseModel):\n    type: str\n    message: str\n\n\n@hatchet.task(input_validator=WebhookInput, on_events=["webhook:test"])\ndef webhook(input: WebhookInput, ctx: Context) -> dict[str, str]:\n    return input.model_dump()\n\n\ndef main() -> None:\n    worker = hatchet.worker("webhook-worker", workflows=[webhook])\n    worker.start()\n\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/webhooks/worker.py',
  blocks: {
    webhooks: {
      start: 2,
      stop: 24,
    },
  },
  highlights: {},
};

export default snippet;
