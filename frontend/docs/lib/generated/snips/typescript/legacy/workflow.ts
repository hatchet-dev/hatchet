import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { Workflow } from \'@hatchet-dev/typescript-sdk/workflow\';\n\nexport const simple: Workflow = {\n  id: \'legacy-workflow\',\n  description: \'test\',\n  on: {\n    event: \'user:create\',\n  },\n  steps: [\n    {\n      name: \'step1\',\n      run: async (ctx) => {\n        const input = ctx.workflowInput();\n\n        return { step1: `original input: ${input.Message}` };\n      },\n    },\n    {\n      name: \'step2\',\n      parents: [\'step1\'],\n      run: (ctx) => {\n        const step1Output = ctx.stepOutput(\'step1\');\n\n        return { step2: `step1 output: ${step1Output.step1}` };\n      },\n    },\n  ],\n};\n',
  'source': 'out/typescript/legacy/workflow.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
