import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/workflow';\nimport { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n  GroupKey: string;\n};\n\ntype SimpleOutput = {\n  'to-lower': {\n    TransformedMessage: string;\n  };\n};\n\nconst sleep = (ms: number) =>\n  new Promise((resolve) => {\n    setTimeout(resolve, ms);\n  });\n\n// > Concurrency Strategy With Key\nexport const multiConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple-concurrency',\n  concurrency: [\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.GroupKey',\n    },\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.UserId',\n    },\n  ],\n});\n\nmultiConcurrency.task({\n  name: 'to-lower',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
  "source": "out/typescript/multiple_wf_concurrency/workflow.ts",
  "blocks": {
    "concurrency_strategy_with_key": {
      "start": 21,
      "stop": 35
    }
  },
  "highlights": {}
};

export default snippet;
