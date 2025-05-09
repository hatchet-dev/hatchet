import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\n\n// > On Failure Task\nexport const failureWorkflow = hatchet.workflow({\n  name: 'always-fail',\n});\n\nfailureWorkflow.task({\n  name: 'always-fail',\n  fn: async () => {\n    throw new Error('intentional failure');\n  },\n});\n\nfailureWorkflow.onFailure({\n  name: 'on-failure',\n  fn: async (input, ctx) => {\n    console.log('onFailure for run:', ctx.workflowRunId());\n    return {\n      'on-failure': 'success',\n    };\n  },\n});\n",
  source: 'out/typescript/on_failure/workflow.ts',
  blocks: {
    on_failure_task: {
      start: 4,
      stop: 23,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
