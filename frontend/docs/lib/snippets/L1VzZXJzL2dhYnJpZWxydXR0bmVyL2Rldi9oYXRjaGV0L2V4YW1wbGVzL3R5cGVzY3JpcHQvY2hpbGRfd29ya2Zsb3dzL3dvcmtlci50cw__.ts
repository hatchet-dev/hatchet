// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/child_workflows/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { parent, child } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('child-workflow-worker', {\n    workflows: [parent, child],\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
