import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import asyncio\nimport random\n\nfrom examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\n\nasync def main() -> None:\n    hatchet = Hatchet()\n\n    # Generate a random stream key to use to track all\n    # stream events for this workflow run.\n\n    streamKey = "streamKey"\n    streamVal = f"sk-{random.randint(1, 100)}"\n\n    # Specify the stream key as additional metadata\n    # when running the workflow.\n\n    # This key gets propagated to all child workflows\n    # and can have an arbitrary property name.\n    bulk_parent_wf.run(\n        input=ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={streamKey: streamVal}),\n    )\n\n    # Stream all events for the additional meta key value\n    listener = hatchet.listener.stream_by_additional_metadata(streamKey, streamVal)\n\n    async for event in listener:\n        print(event.type, event.payload)\n\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
  source: 'out/python/bulk_fanout/stream.py',
  blocks: {},
  highlights: {},
};

export default snippet;
