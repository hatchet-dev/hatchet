import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// > Route tasks to workers with matching labels\nexport const simple = hatchet.task({\n  name: \'simple\',\n  desiredWorkerLabels: {\n    cpu: {\n      value: \'2x\',\n    },\n  },\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nhatchet.worker(\'task-routing-worker\', {\n  workflows: [simple],\n  labels: {\n    cpu: process.env.CPU_LABEL,\n  },\n});\n\n',
  'source': 'out/typescript/landing_page/task-routing.ts',
  'blocks': {
    'route_tasks_to_workers_with_matching_labels': {
      'start': 9,
      'stop': 28
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
