import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "# > Simple\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\nfrom hatchet_sdk.v0 import Hatchet as HatchetV0\n\nhatchet = Hatchet(debug=True)\n# hatchet_v0 = HatchetV0(debug=True)\n\n\n@hatchet.task()\ndef simple(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    return {\"result\": \"Hello, world!\"}\n\n\n@hatchet.durable_task()\ndef simple_durable(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    return {\"result\": \"Hello, world!\"}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", workflows=[simple, simple_durable])\n    worker.start()\n\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/simple/worker.py",
  "blocks": {
    "simple": {
      "start": 2,
      "stop": 24
    }
  },
  "highlights": {}
};

export default snippet;
