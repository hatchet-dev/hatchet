import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from hatchet_sdk import Context\nfrom pydantic import BaseModel\n\nfrom ..hatchet_client import hatchet\n\n\nclass SimpleInput(BaseModel):\n    message: str\n\n\nclass SimpleOutput(BaseModel):\n    transformed_message: str\n\n\n# Declare the task to run\n@hatchet.task(name='first-task')\ndef first_task(input: SimpleInput, ctx: Context) -> SimpleOutput:\n    print('first-task task called')\n\n    return SimpleOutput(transformed_message=input.message.lower())\n",
  source: 'out/python/quickstart/workflows/first_task.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
