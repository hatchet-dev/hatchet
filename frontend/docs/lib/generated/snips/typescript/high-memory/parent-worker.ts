import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "// ❓ Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { parent } from './workflow-with-child';\n\nasync function main() {\n  const worker = await hatchet.worker('parent-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [parent],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 20,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  "source": "out/typescript/high-memory/parent-worker.ts",
  "blocks": {},
  "highlights": {}
};

export default snippet;
