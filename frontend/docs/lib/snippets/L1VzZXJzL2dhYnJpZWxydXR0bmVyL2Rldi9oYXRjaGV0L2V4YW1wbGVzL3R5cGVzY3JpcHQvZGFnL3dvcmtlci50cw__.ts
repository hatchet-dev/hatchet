// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/dag/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { dag } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('dag-worker', {\n    workflows: [dag],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
