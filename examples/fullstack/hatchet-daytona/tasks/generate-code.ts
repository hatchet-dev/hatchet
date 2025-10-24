import { hatchet } from '../clients';
import { openai } from '@ai-sdk/openai';
import { generateText } from 'ai';

export type GenerateCodeInput = {
  prompt: string;
};

export const generate = hatchet.task({
  name: 'generate',
  retries: 3,
  fn: async (input: GenerateCodeInput) => {
    const { text: code } = await generateText({
      model: openai('gpt-4o-mini'),
      messages: [
        {
          role: 'system',
          content: 'You are a Python code generator. Generate only pure Python code that satisfies the user\'s request. Do not include any explanations, markdown formatting, or code blocks - just return the raw Python code that can be executed directly.'
        },
        {
          role: 'user',
          content: input.prompt
        }
      ],
      temperature: 0.7,
    });

    return {
      code
    };
  },
});
