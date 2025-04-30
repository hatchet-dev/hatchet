import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { ConcurrencyLimitStrategy } from \'@hatchet-dev/typescript-sdk/workflow\';\nimport { hatchet } from \'../hatchet-client\';\n\ntype SimpleInput = {\n  Message: string;\n  GroupKey: string;\n};\n\ntype SimpleOutput = {\n  \'to-lower\': {\n    TransformedMessage: string;\n  };\n};\n\nconst sleep = (ms: number) =>\n  new Promise((resolve) => {\n    setTimeout(resolve, ms);\n  });\n\n// > Concurrency Strategy With Key\nexport const simpleConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: \'simple-concurrency\',\n  concurrency: {\n    maxRuns: 1,\n    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    expression: \'input.GroupKey\',\n  },\n});\n\nsimpleConcurrency.task({\n  name: \'to-lower\',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// > Multiple Concurrency Keys\nexport const multipleConcurrencyKeys = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: \'simple-concurrency\',\n  concurrency: [\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: \'input.Tier\',\n    },\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: \'input.Account\',\n    },\n  ],\n});\n\nmultipleConcurrencyKeys.task({\n  name: \'to-lower\',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n',
  'source': 'out/typescript/concurrency-rr/workflow.ts',
  'blocks': {
    'concurrency_strategy_with_key': {
      'start': 21,
      'stop': 28
    },
    'multiple_concurrency_keys': {
      'start': 41,
      'stop': 55
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
