// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/priority/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { priorityTasks } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('priority-worker', {\n    workflows: [...priorityTasks],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
