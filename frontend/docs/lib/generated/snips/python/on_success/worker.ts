import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\non_success_workflow = hatchet.workflow(name=\'OnSuccessWorkflow\')\n\n\n@on_success_workflow.task()\ndef first_task(input: EmptyModel, ctx: Context) -> None:\n    print(\'First task completed successfully\')\n\n\n@on_success_workflow.task(parents=[first_task])\ndef second_task(input: EmptyModel, ctx: Context) -> None:\n    print(\'Second task completed successfully\')\n\n\n@on_success_workflow.task(parents=[first_task, second_task])\ndef third_task(input: EmptyModel, ctx: Context) -> None:\n    print(\'Third task completed successfully\')\n\n\n@on_success_workflow.task()\ndef fourth_task(input: EmptyModel, ctx: Context) -> None:\n    print(\'Fourth task completed successfully\')\n\n\n@on_success_workflow.on_success_task()\ndef on_success_task(input: EmptyModel, ctx: Context) -> None:\n    print(\'On success task completed successfully\')\n\n\ndef main() -> None:\n    worker = hatchet.worker(\'on-success-worker\', workflows=[on_success_workflow])\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/on_success/worker.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
