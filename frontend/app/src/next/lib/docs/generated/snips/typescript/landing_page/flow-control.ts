import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/protoc/v1/workflows';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// > Process what you can handle\nexport const simple = hatchet.task({\n  name: 'simple',\n  concurrency: {\n    expression: 'input.user_id',\n    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    maxRuns: 1,\n  },\n  rateLimits: [\n    {\n      key: 'api_throttle',\n      units: 1,\n    },\n  ],\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
  source: 'out/typescript/landing_page/flow-control.ts',
  blocks: {
    process_what_you_can_handle: {
      start: 10,
      stop: 28,
    },
  },
  highlights: {},
};

export default snippet;
