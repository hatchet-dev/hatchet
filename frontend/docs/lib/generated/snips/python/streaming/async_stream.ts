import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\n\nfrom examples.streaming.worker import hatchet, stream_task\nfrom hatchet_sdk.clients.listeners.run_event_listener import StepRunEventType\n\n\nasync def main() -> None:\n    # > Consume\n    ref = await stream_task.aio_run_no_wait()\n\n    async for chunk in hatchet.subscribe_to_stream(ref.workflow_run_id):\n        print(chunk, flush=True, end=\"\")\n\n\nif __name__ == \"__main__\":\n    asyncio.run(main())\n",
  "source": "out/python/streaming/async_stream.py",
  "blocks": {
    "consume": {
      "start": 9,
      "stop": 12
    }
  },
  "highlights": {}
};

export default snippet;
