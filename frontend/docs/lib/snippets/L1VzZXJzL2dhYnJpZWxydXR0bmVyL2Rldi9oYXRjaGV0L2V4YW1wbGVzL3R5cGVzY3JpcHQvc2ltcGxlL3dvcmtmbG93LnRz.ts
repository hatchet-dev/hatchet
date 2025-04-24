// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/simple/workflow.ts
export const content = "// ❓ Declaring a Task\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\nexport const simple = hatchet.task({\n  name: 'simple',\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// !!\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n";
export const language = "ts";
