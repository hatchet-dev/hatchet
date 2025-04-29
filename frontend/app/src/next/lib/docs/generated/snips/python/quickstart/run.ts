import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import asyncio\n\nfrom .workflows.first_task import first_task, SimpleInput\n\n\nasync def main() -> None:\n    result = await first_task.aio_run(SimpleInput(Message='Hello World!'))\n\n    print(\n        'Finished running task, and got the transformed message! The transformed message is:',\n        result.transformed_message,\n    )\n\n\nif __name__ == '__main__':\n    asyncio.run(main())\n",
  source: 'out/python/quickstart/run.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
