// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/non_retryable/run.ts
export const content = "import { nonRetryableWorkflow } from './workflow';\n\nasync function main() {\n  const res = await nonRetryableWorkflow.runNoWait({});\n\n  // eslint-disable-next-line no-console\n  console.log(res);\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
