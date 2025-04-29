import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from hatchet_sdk import DurableContext, EmptyModel, Hatchet, UserEventCondition\n\nhatchet = Hatchet(debug=True)\n\nEVENT_KEY = \'user:update\'\n\n\n# > Durable Event\n@hatchet.durable_task(name=\'DurableEventTask\')\nasync def durable_event_task(input: EmptyModel, ctx: DurableContext) -> None:\n    res = await ctx.aio_wait_for(\n        \'event\',\n        UserEventCondition(event_key=\'user:update\'),\n    )\n\n    print(\'got event\', res)\n\n\n\n\n\n@hatchet.durable_task(name=\'DurableEventWithFilterTask\')\nasync def durable_event_task_with_filter(\n    input: EmptyModel, ctx: DurableContext\n) -> None:\n    # > Durable Event With Filter\n    res = await ctx.aio_wait_for(\n        \'event\',\n        UserEventCondition(\n            event_key=\'user:update\', expression=\'input.user_id == \'1234\'\'\n        ),\n    )\n    \n\n    print(\'got event\', res)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'durable-event-worker\',\n        workflows=[durable_event_task, durable_event_task_with_filter],\n    )\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/durable_event/worker.py',
  'blocks': {
    'durable_event': {
      'start': 9,
      'stop': 18
    },
    'durable_event_with_filter': {
      'start': 26,
      'stop': 31
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
