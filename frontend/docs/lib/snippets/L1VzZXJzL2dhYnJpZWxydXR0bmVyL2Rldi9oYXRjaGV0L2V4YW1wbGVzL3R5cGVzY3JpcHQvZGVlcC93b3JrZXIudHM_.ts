// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/deep/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { parent, child1, child2, child3, child4, child5 } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    workflows: [parent, child1, child2, child3, child4, child5],\n    slots: 5000,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
