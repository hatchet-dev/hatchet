import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from datetime import timedelta\n\nfrom hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, UserEventCondition\n\nhatchet = Hatchet(debug=True)\n\n# > Create a durable workflow\ndurable_workflow = hatchet.workflow(name=\'DurableWorkflow\')\n\n\nephemeral_workflow = hatchet.workflow(name=\'EphemeralWorkflow\')\n\n\n# > Add durable task\nEVENT_KEY = \'durable-example:event\'\nSLEEP_TIME = 5\n\n\n@durable_workflow.task()\nasync def ephemeral_task(input: EmptyModel, ctx: Context) -> None:\n    print(\'Running non-durable task\')\n\n\n@durable_workflow.durable_task()\nasync def durable_task(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:\n    print(\'Waiting for sleep\')\n    await ctx.aio_sleep_for(duration=timedelta(seconds=SLEEP_TIME))\n    print(\'Sleep finished\')\n\n    print(\'Waiting for event\')\n    await ctx.aio_wait_for(\n        \'event\',\n        UserEventCondition(event_key=EVENT_KEY, expression=\'true\'),\n    )\n    print(\'Event received\')\n\n    return {\n        \'status\': \'success\',\n    }\n\n\n\n\n@ephemeral_workflow.task()\ndef ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:\n    print(\'Running non-durable task\')\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'durable-worker\', workflows=[durable_workflow, ephemeral_workflow]\n    )\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/durable/worker.py',
  'blocks': {
    'create_a_durable_workflow': {
      'start': 8,
      'stop': 8
    },
    'add_durable_task': {
      'start': 15,
      'stop': 41
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
