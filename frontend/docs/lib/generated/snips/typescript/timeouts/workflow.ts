import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '// > Declaring a Task\nimport sleep from \'@hatchet-dev/typescript-sdk/util/sleep\';\nimport { hatchet } from \'../hatchet-client\';\n\n// (optional) Define the input type for the workflow\nexport const cancellation = hatchet.task({\n  name: \'cancellation\',\n  executionTimeout: \'3s\',\n  fn: async (_, { cancelled }) => {\n    await sleep(10 * 1000);\n\n    if (cancelled) {\n      throw new Error(\'Task was cancelled\');\n    }\n\n    return {\n      Completed: true,\n    };\n  },\n});\n\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n',
  'source': 'out/typescript/timeouts/workflow.ts',
  'blocks': {
    'declaring_a_task': {
      'start': 2,
      'stop': 20
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
