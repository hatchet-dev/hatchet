import * as z from 'zod';
import { BaseWorkflowDeclaration, InputType, OutputType } from '@hatchet/v1';
import type { FunctionTool } from '@openai/agents';

export type OpenAIToolFuncT = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => FunctionTool;

export const OpenAIToolFunc = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => {
  try {
    require.resolve('@openai/agents');
    require.resolve('zod-to-json-schema');
  } catch {
    throw new Error(
      "To use Hatchet's OpenAI integration, you must install the @openai/agents and zod-to-json-schema packages: npm install @openai/agents zod-to-json-schema"
    );
  }
  if (!runnable.definition.inputValidator) {
    throw new Error('inputValidator must be defined');
  }
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { tool } = require('@openai/agents');
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const inputValidator = runnable.definition.inputValidator! as z.ZodObject<any>;
  const { description } = runnable.definition;
  if (description === undefined) {
    throw new Error('Runnable description must be defined');
  }
  return tool({
    name: runnable.name,
    description: description,
    parameters: inputValidator.toJSONSchema(),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    execute: async (input: any): Promise<string> => {
      const result = await runnable.run(input);
      return JSON.stringify(result);
    },
  });
};
