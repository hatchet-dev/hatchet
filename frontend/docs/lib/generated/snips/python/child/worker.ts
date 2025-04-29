import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': '# > Simple\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\nclass SimpleInput(BaseModel):\n    message: str\n\n\nclass SimpleOutput(BaseModel):\n    transformed_message: str\n\n\nchild_task = hatchet.workflow(name=\'SimpleWorkflow\', input_validator=SimpleInput)\n\n\n@child_task.task(name=\'step1\')\ndef step1(input: SimpleInput, ctx: Context) -> SimpleOutput:\n    print(\'executed step1: \', input.message)\n    return SimpleOutput(transformed_message=input.message.upper())\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\'test-worker\', slots=1, workflows=[child_task])\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/child/worker.py',
  'blocks': {
    'simple': {
      'start': 2,
      'stop': 26
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
