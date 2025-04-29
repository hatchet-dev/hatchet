import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '// ❓ Declaring a Worker\nimport { hatchet } from \'../hatchet-client\';\nimport { child } from \'./workflow-with-child\';\n\nasync function main() {\n  const worker = await hatchet.worker(\'child-worker\', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [child],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 1000,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/high-memory/child-worker.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
