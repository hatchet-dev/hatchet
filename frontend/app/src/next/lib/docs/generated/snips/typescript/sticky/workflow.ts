import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { StickyStrategy } from '@hatchet-dev/typescript-sdk/protoc/workflows';\nimport { hatchet } from '../hatchet-client';\nimport { child } from '../child_workflows/workflow';\n\n// > Sticky Task\nexport const sticky = hatchet.task({\n  name: 'sticky',\n  retries: 3,\n  sticky: StickyStrategy.SOFT,\n  fn: async (_, ctx) => {\n    // specify a child workflow to run on the same worker\n    const result = await ctx.runChild(\n      child,\n      {\n        N: 1,\n      },\n      { sticky: true }\n    );\n\n    return {\n      result,\n    };\n  },\n});\n",
  source: 'out/typescript/sticky/workflow.ts',
  blocks: {
    sticky_task: {
      start: 6,
      stop: 24,
    },
  },
  highlights: {},
};

export default snippet;
