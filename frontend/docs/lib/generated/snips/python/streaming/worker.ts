import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\nimport re\nfrom typing import Generator\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# > Streaming\n\ncontent = \"\"\"\nHappy families are all alike; every unhappy family is unhappy in its own way.\n\nEverything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him. This position of affairs had now lasted three days, and not only the husband and wife themselves, but all the members of their family and household, were painfully conscious of it. Every person in the house felt that there was so sense in their living together, and that the stray people brought together by chance in any inn had more in common with one another than they, the members of the family and household of the Oblonskys. The wife did not leave her own room, the husband had not been at home for three days. The children ran wild all over the house; the English governess quarreled with the housekeeper, and wrote to a friend asking her to look out for a new situation for her; the man-cook had walked off the day before just at dinner time; the kitchen-maid, and the coachman had given warning.\n\"\"\"\n\n\ndef create_chunks(content: str, n: int) -> Generator[str, None]:\n    for i in range(0, len(content), n):\n        yield content[i : i + n]\n\n\nchunks = list(create_chunks(content, 10))\n\n\n@hatchet.task()\nasync def stream_task(input: EmptyModel, ctx: Context) -> None:\n    for chunk in chunks:\n        ctx.put_stream(chunk)\n        await asyncio.sleep(0.025)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", workflows=[stream_task])\n    worker.start()\n\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/streaming/worker.py",
  "blocks": {
    "streaming": {
      "start": 10,
      "stop": 37
    }
  },
  "highlights": {}
};

export default snippet;
