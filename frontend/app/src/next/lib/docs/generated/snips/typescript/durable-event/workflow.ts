import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '// import sleep from \'@hatchet-dev/typescript-sdk/util/sleep\';\nimport { hatchet } from \'../hatchet-client\';\n\n// > Durable Event\nexport const durableEvent = hatchet.durableTask({\n  name: \'durable-event\',\n  executionTimeout: \'10m\',\n  fn: async (_, ctx) => {\n    const res = ctx.waitFor({\n      eventKey: \'user:update\',\n    });\n\n    console.log(\'res\', res);\n\n    return {\n      Value: \'done\',\n    };\n  },\n});\n\n\nexport const durableEventWithFilter = hatchet.durableTask({\n  name: \'durable-event-with-filter\',\n  executionTimeout: \'10m\',\n  fn: async (_, ctx) => {\n    // > Durable Event With Filter\n    const res = ctx.waitFor({\n      eventKey: \'user:update\',\n      expression: \'input.userId == \'1234\'\',\n    });\n    \n\n    console.log(\'res\', res);\n\n    return {\n      Value: \'done\',\n    };\n  },\n});\n\n',
  'source': 'out/typescript/durable-event/workflow.ts',
  'blocks': {
    'durable_event': {
      'start': 5,
      'stop': 19
    },
    'durable_event_with_filter': {
      'start': 26,
      'stop': 29
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
