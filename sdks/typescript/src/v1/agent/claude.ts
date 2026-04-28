import * as z from 'zod';
import { BaseWorkflowDeclaration, InputType, OutputType } from '@hatchet/v1';
import type { SdkMcpToolDefinition } from '@anthropic-ai/claude-agent-sdk';
import type { CallToolResult, ToolAnnotations } from '@modelcontextprotocol/sdk/types.js';

export type ClaudeToolFuncT = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>,
  annotations?: ToolAnnotations
) => SdkMcpToolDefinition;

export const ClaudeToolFunc = <I extends InputType, O extends OutputType>(
  runnable: BaseWorkflowDeclaration<I, O>,
  annotations?: ToolAnnotations
) => {
  try {
    require.resolve('@anthropic-ai/claude-agent-sdk');
  } catch {
    throw new Error(
      "To use Hatchet's Claude integration, you must install the @anthropic-ai/claude-agent-sdk and @modelcontextprotocol/sdk packages: npm install @anthropic-ai/claude-agent-sdk @modelcontextprotocol/sdk"
    );
  }
  // Check Zod v4 is installed before using any v4-specific schema APIs.
  if (!('_zod' in z.string())) {
    throw new Error(
      "To use Hatchet's Claude agent SDK integration, Zod v4 must be installed. " +
        'Please upgrade: npm install zod@^4.0.0'
    );
  }
  if (!runnable.definition.inputValidator) {
    throw new Error('inputValidator must be defined');
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const inputValidatorV4 = runnable.definition.inputValidator as unknown as z.ZodObject<any>;
  const { description } = runnable.definition;
  if (description === undefined) {
    throw new Error('Runnable description must be defined');
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handler = async (args: any, _: unknown): Promise<CallToolResult> => {
    const result = await runnable.run(args);
    return {
      content: [{ type: 'text', text: JSON.stringify(result) }],
    };
  };
  return {
    annotations: annotations,
    description: description,
    handler: handler,
    name: runnable.name,
    inputSchema: inputValidatorV4.shape,
  };
};
