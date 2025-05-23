import { hatchet } from '@/hatchet.client';
import { z } from 'zod';
import { generateObject } from 'ai';
import { openai } from '@ai-sdk/openai';

const JudgeResultsInputSchema = z.object({
  query: z.string(),
  result: z.string(),
});

type JudgeResultsInput = z.infer<typeof JudgeResultsInputSchema>;

type JudgeResultsOutput = {   
  reason: string;
  isComplete: boolean;
};

export const judgeResults = hatchet.task({
  name: 'judge-results',
  fn: async (input: JudgeResultsInput): Promise<JudgeResultsOutput> => {
    const validatedInput = JudgeResultsInputSchema.parse(input);

    const result = await generateObject({
      prompt: `
Judge the following answer to the query for completeness: 
"""${validatedInput.query}"""

Answer: 
"""${validatedInput.result}"""

Completeness means that the answer includes all the information that is relevant to the query and that the answer is not missing any important details. Does the answer leave any new questions unanswered?
`,
      model: openai('gpt-4.1-mini'),
      schema: z.object({
        reason: z.string(),
        isComplete: z.boolean(),
      }),
    });

    return {
      reason: result.object.reason,
      isComplete: result.object.isComplete,
    };
  },
});
