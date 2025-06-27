import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\n\nfrom examples.streaming.worker import stream_task\nfrom hatchet_sdk.clients.listeners.run_event_listener import StepRunEventType\n\n\nasync def main() -> None:\n    # > Consume\n    ref = await stream_task.aio_run_no_wait()\n\n    async for chunk in ref.stream():\n        if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:\n            print(chunk.payload, flush=True, end=\"\")\n\n\nif __name__ == \"__main__\":\n    asyncio.run(main())\n",
  "source": "out/python/streaming/async_stream.py",
  "blocks": {
    "consume": {
      "start": 9,
      "stop": 13
    }
  },
  "highlights": {}
};

export default snippet;
