import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n\nclass Input(EmptyModel):\n    index: int\n\n\n@hatchet.task(input_validator=Input)\nasync def return_exceptions_task(input: Input, ctx: Context) -> dict[str, str]:\n    if input.index % 2 == 0:\n        raise ValueError(f\"error in task with index {input.index}\")\n\n    return {\"message\": \"this is a successful task.\"}\n",
  "source": "out/python/return_exceptions/worker.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
