import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\n\nexport type Input = {\n  Message: string;\n};\n\ntype OnCronOutput = {\n  job: {\n    TransformedMessage: string;\n  };\n};\n\n// > Run Workflow on Cron\nexport const onCron = hatchet.workflow<Input, OnCronOutput>({\n  name: \'on-cron-workflow\',\n  on: {\n    // 👀 add a cron expression to run the workflow every 15 minutes\n    cron: \'*/15 * * * *\',\n  },\n});\n\nonCron.task({\n  name: \'job\',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n',
  'source': 'out/typescript/on_cron/workflow.ts',
  'blocks': {
    'run_workflow_on_cron': {
      'start': 14,
      'stop': 20
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
