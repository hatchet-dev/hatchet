// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/inferred-typing/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { declaredType, inferredType, inferredTypeDurable, crazyWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    workflows: [declaredType, inferredType, inferredTypeDurable, crazyWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
