// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/dag_match_condition/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { dagWithConditions } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('dag-worker', {\n    workflows: [dagWithConditions],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
