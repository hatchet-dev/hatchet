import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from hatchet_sdk import (\n    Context,\n    EmptyModel,\n    Hatchet,\n    StickyStrategy,\n    TriggerWorkflowOptions,\n)\n\nhatchet = Hatchet(debug=True)\n\n# > StickyWorker\n\n\nsticky_workflow = hatchet.workflow(\n    name=\'StickyWorkflow\',\n    # 👀 Specify a sticky strategy when declaring the workflow\n    sticky=StickyStrategy.SOFT,\n)\n\n\n@sticky_workflow.task()\ndef step1a(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {\'worker\': ctx.worker.id()}\n\n\n@sticky_workflow.task()\ndef step1b(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {\'worker\': ctx.worker.id()}\n\n\n\n\n# > StickyChild\n\nsticky_child_workflow = hatchet.workflow(\n    name=\'StickyChildWorkflow\', sticky=StickyStrategy.SOFT\n)\n\n\n@sticky_workflow.task(parents=[step1a, step1b])\nasync def step2(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    ref = await sticky_child_workflow.aio_run_no_wait(\n        options=TriggerWorkflowOptions(sticky=True)\n    )\n\n    await ref.aio_result()\n\n    return {\'worker\': ctx.worker.id()}\n\n\n@sticky_child_workflow.task()\ndef child(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {\'worker\': ctx.worker.id()}\n\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'sticky-worker\', slots=10, workflows=[sticky_workflow, sticky_child_workflow]\n    )\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/sticky_workers/worker.py',
  'blocks': {
    'stickyworker': {
      'start': 12,
      'stop': 30
    },
    'stickychild': {
      'start': 33,
      'stop': 54
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
