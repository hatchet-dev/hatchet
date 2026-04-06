try {
  require.resolve('@openai/agents');
  require.resolve('zod-to-json-schema');
} catch {
  throw new Error(
    "To use Hatchet's OpenAI integration, you must install the @openai/agents and zod-to-json-schema packages: npm install @openai/agents zod-to-json-schema"
  );
}
import * as z from 'zod';
import { BaseWorkflowDeclaration, InputType, OutputType } from '@hatchet/v1';
import type { FunctionTool } from '@openai/agents';
import { zodToJsonSchema } from 'zod-to-json-schema';

export type OpenAIToolFuncT = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => FunctionTool;

export const OpenAIToolFunc = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => {
  if (!runnable.definition.inputValidator) {
    throw new Error('inputValidator must be defined');
  }
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { tool } = require('@openai/agents');
  const inputValidator = runnable.definition.inputValidator! as z.ZodObject<any>;
  const { description } = runnable.definition;
  if (description === undefined) {
    throw new Error('Runnable description must be defined');
  }
  return tool({
    name: runnable.name,
    description: description,
    // @ts-expect-error TS2589
    parameters: zodToJsonSchema(inputValidator, {
      $refStrategy: 'none',
    }),
    execute: async (input: any): Promise<string> => {
      const result = await runnable.run(input);
      return JSON.stringify(result);
    },
  });
};
