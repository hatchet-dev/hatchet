import * as z from 'zod/v4';
import { BaseWorkflowDeclaration, InputType, OutputType } from '@hatchet/v1';
import type { FunctionTool } from '@openai/agents';

export type OpenAIToolFuncT = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => FunctionTool;

export const OpenAIToolFunc = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>
) => {
  // Check both requirements before loading @openai/agents — it crashes at load time with Zod v3.
  // z is imported from 'zod', so '_zod' in z.string() reflects the user's actual installed version.
  const hasZodV4 = '_zod' in z.string();
  let hasOpenAIAgents = true;
  try {
    require.resolve('@openai/agents');
  } catch {
    hasOpenAIAgents = false;
  }
  if (!hasZodV4 || !hasOpenAIAgents) {
    throw new Error(
      "To use Hatchet's OpenAI agent SDK integration, you must install Zod v4 and @openai/agents: npm install zod@^4.0.0 @openai/agents"
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
    parameters: z.toJSONSchema(inputValidatorV4),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    execute: async (input: any): Promise<string> => {
      const result = await runnable.run(input);
      return JSON.stringify(result);
    },
  });
};
