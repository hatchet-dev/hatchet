import { client } from './../client';
import z from 'zod';

const CommonAgentResponseSchema = z.object({
  message: z.string(),
});

const supportAgent = client.agent({
  name: 'support-agent',
  executionTimeout: '1m',
  inputSchema: z.object({
    message: z.string(),
  }),
  outputSchema: CommonAgentResponseSchema,
  description: 'A support agent that provides support to the user',
  fn: async (input, ctx) => {
    return { message: 'Hello from support agent' };
  },
});

const salesAgent = client.agent({
  name: 'sales-agent',
  description: 'A sales agent that sells the product to the user',
  executionTimeout: '1m',
  inputSchema: z.object({
    message: z.string(),
  }),
  outputSchema: CommonAgentResponseSchema,
  fn: async (input, ctx) => {
    return { message: 'Hello from sales agent' };
  },
});

export const multiAgentToolbox = client.toolbox({
  tools: [supportAgent, salesAgent],
});

export const rootAgent = client.agent({
  name: 'root-agent',
  executionTimeout: '1m',
  inputSchema: z.object({
    message: z.string(),
  }),
  outputSchema: z.object({
    message: z.string(),
  }),
  description: 'A root agent that orchestrates the other agents',
  fn: async (input, ctx) => {
    const result = await multiAgentToolbox.pickAndRun({
      prompt: input.message,
    });

    return { message: result.output.message };
  },
});
