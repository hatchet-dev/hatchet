import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '// > Declaring a Task\nimport sleep from \'@hatchet-dev/typescript-sdk/util/sleep\';\nimport { hatchet } from \'../hatchet-client\';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\nexport const simple = hatchet.task({\n  name: \'simple\',\n  retries: 3,\n  fn: async (input: SimpleInput, ctx) => {\n    ctx.log(\'hello from the workflow\');\n    await sleep(100);\n    ctx.log(\'goodbye from the workflow\');\n    await sleep(100);\n    if (ctx.retryCount() < 2) {\n      throw new Error(\'test error\');\n    }\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n',
  'source': 'out/typescript/simple/workflow.ts',
  'blocks': {
    'declaring_a_task': {
      'start': 2,
      'stop': 26
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
