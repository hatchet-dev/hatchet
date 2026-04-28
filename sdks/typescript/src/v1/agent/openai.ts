import * as z from 'zod';
import { BaseWorkflowDeclaration, InputType, OutputType } from '@hatchet/v1';
import type { FunctionTool } from '@openai/agents';

export type OpenAIToolFuncT = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => FunctionTool;

export const OpenAIToolFunc = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => {
  // Check Zod v4 is installed BEFORE requiring @openai/agents — it crashes at load time with Zod v3.
  // z is imported from 'zod', so z.string()._zod reflects the user's actual installed version.
  if (!('_zod' in z.string())) {
    throw new Error(
      "To use Hatchet's OpenAI agent SDK integration, Zod v4 must be installed. " +
        'Please upgrade: npm install zod@^4.0.0'
    );
  }
  try {
    require.resolve('@openai/agents');
  } catch {
    throw new Error(
      "To use Hatchet's OpenAI integration, you must install the @openai/agents package: npm install @openai/agents"
    );
  }
  if (!runnable.definition.inputValidator) {
    throw new Error('inputValidator must be defined');
  }
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { tool } = require('@openai/agents');
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const inputValidatorV4 = runnable.definition.inputValidator as unknown as z.ZodObject<any>;
  const { description } = runnable.definition;
  if (description === undefined) {
    throw new Error('Runnable description must be defined');
  }
  return tool({
    name: runnable.name,
    description: description,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    parameters: (inputValidatorV4 as any).toJSONSchema(),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    execute: async (input: any): Promise<string> => {
      const result = await runnable.run(input);
      return JSON.stringify(result);
    },
  });
};
