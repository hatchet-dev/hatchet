import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '// > Declaring an External Workflow Reference\nimport { hatchet } from \'../hatchet-client\';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// (optional) Define the output type for the workflow\nexport type SimpleOutput = {\n  \'to-lower\': {\n    TransformedMessage: string;\n  };\n};\n\n// declare the workflow with the same name as the\n// workflow name on the worker\nexport const simple = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: \'simple\',\n});\n\n// you can use all the same run methods on the stub\n// with full type-safety\nsimple.run({ Message: \'Hello, World!\' });\nsimple.runNoWait({ Message: \'Hello, World!\' });\nsimple.schedule(new Date(), { Message: \'Hello, World!\' });\nsimple.cron(\'my-cron\', \'0 0 * * *\', { Message: \'Hello, World!\' });\n\n',
  'source': 'out/typescript/simple/stub-workflow.ts',
  'blocks': {
    'declaring_an_external_workflow_reference': {
      'start': 2,
      'stop': 27
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
