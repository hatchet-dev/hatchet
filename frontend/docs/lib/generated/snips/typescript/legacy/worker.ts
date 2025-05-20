import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('legacy-worker', {\n    workflows: [simple],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  "source": "out/typescript/legacy/worker.ts",
  "blocks": {},
  "highlights": {}
};

export default snippet;
