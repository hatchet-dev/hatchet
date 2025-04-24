// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/retries/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { retries } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-fail-worker', {\n    workflows: [retries],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
