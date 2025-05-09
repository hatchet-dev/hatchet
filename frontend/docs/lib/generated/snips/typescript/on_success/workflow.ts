import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\n\n// > On Success DAG\nexport const onSuccessDag = hatchet.workflow({\n  name: \'on-success-dag\',\n});\n\nonSuccessDag.task({\n  name: \'always-succeed\',\n  fn: async () => {\n    return {\n      \'always-succeed\': \'success\',\n    };\n  },\n});\nonSuccessDag.task({\n  name: \'always-succeed2\',\n  fn: async () => {\n    return {\n      \'always-succeed\': \'success\',\n    };\n  },\n});\n\n// 👀 onSuccess handler will run if all tasks in the workflow succeed\nonSuccessDag.onSuccess({\n  fn: (_, ctx) => {\n    console.log(\'onSuccess for run:\', ctx.workflowRunId());\n    return {\n      \'on-success\': \'success\',\n    };\n  },\n});\n',
  'source': 'out/typescript/on_success/workflow.ts',
  'blocks': {
    'on_success_dag': {
      'start': 4,
      'stop': 33
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
