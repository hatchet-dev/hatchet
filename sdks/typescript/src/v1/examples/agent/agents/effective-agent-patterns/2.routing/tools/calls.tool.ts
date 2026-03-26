import { client } from './../../../../client';
import z from 'zod';
import { generateText } from 'ai';

const CallInput = z.object({
  message: z.string(),
});

const CallOutput = z.object({
  response: z.string(),
});

export const supportTool = client.tool({
  name: 'support-tool',
  description: 'A tool that provides technical support for the user',
  inputSchema: CallInput,
  outputSchema: CallOutput,
  fn: async (input) => {
    const response = await generateText({
      model: client.defaultLanguageModel,
      prompt: `You are a support agent. The answer is usually to turn it on and off. The user has asked the following question: ${input.message}. Please provide a response to the user.`,
    });

    return {
      response: response.text,
    };
  },
});

export const salesTool = client.tool({
  name: 'sales-tool',
  description: 'A tool that provides sales support for the user',
  inputSchema: CallInput,
  outputSchema: CallOutput,
  fn: async (input) => {
    const response = await generateText({
      model: client.defaultLanguageModel,
      prompt: `You are a sales agent. The product cost is $42.The user has asked the following question: ${input.message}. Please provide a response to the user.`,
    });

    return {
      response: response.text,
    };
  },
});
