// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/on_success/run.ts
export const content = "/* eslint-disable no-console */\nimport { onSuccessDag } from './workflow';\n\nasync function main() {\n  try {\n    const res2 = await onSuccessDag.run({});\n    console.log(res2);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n";
export const language = "ts";
