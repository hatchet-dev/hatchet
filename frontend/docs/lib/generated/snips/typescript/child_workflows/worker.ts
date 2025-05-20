import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { hatchet } from '../hatchet-client';\nimport { parent, child } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('child-workflow-worker', {\n    workflows: [parent, child],\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  "source": "out/typescript/child_workflows/worker.ts",
  "blocks": {},
  "highlights": {}
};

export default snippet;
