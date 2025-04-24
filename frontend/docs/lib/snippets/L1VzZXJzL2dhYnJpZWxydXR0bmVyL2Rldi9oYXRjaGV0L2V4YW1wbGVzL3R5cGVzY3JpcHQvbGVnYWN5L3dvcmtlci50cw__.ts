// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/legacy/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('legacy-worker', {\n    workflows: [simple],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
