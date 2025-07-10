import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\nfrom typing import Generator\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=False)\n\n# > Streaming\n\nanna_karenina = \"\"\"\nHappy families are all alike; every unhappy family is unhappy in its own way.\n\nEverything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.\n\"\"\"\n\n\ndef create_chunks(content: str, n: int) -> Generator[str, None, None]:\n    for i in range(0, len(content), n):\n        yield content[i : i + n]\n\n\nchunks = list(create_chunks(anna_karenina, 10))\n\n\n@hatchet.task()\nasync def stream_task(input: EmptyModel, ctx: Context) -> None:\n    # 👀 Sleeping to avoid race conditions\n    await asyncio.sleep(2)\n\n    for chunk in chunks:\n        await ctx.aio_put_stream(chunk)\n        await asyncio.sleep(0.20)\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", workflows=[stream_task])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/streaming/worker.py",
  "blocks": {
    "streaming": {
      "start": 9,
      "stop": 34
    }
  },
  "highlights": {}
};

export default snippet;
