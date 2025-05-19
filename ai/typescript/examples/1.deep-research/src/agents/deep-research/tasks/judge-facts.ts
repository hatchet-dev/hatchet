import { hatchet } from '@/hatchet.client';
import { z } from 'zod';
import { generateObject } from 'ai';
import { openai } from '@ai-sdk/openai';

const JudgeFactsInputSchema = z.object({
  query: z.string(),
  facts: z.array(z.string()),
});

type JudgeFactsInput = z.infer<typeof JudgeFactsInputSchema>;

type JudgeFactsOutput = {
  hasEnoughFacts: boolean;
  reason: string;
  missingAspects: string[];
};

export const judgeFacts = hatchet.task({
  name: 'judge-facts',
  fn: async (input: JudgeFactsInput): Promise<JudgeFactsOutput> => {
    const validatedInput = JudgeFactsInputSchema.parse(input);

    const result = await generateObject({
      prompt: `
Evaluate if we have enough facts to comprehensively answer this query:
"""${validatedInput.query}"""

Current facts:
${validatedInput.facts.map((fact, i) => `${i + 1}. ${fact}`).join('\n')}

Consider:
1. Are there any key aspects of the query that aren't covered by the current facts?
2. Are the facts diverse enough to provide a complete picture?
3. Are there any gaps in the information that would prevent a comprehensive answer?
4. Are there any technical jargon words that are not defined in the facts that require additional research?
`,
      model: openai('gpt-4.1-mini'),
      schema: z.object({
        hasEnoughFacts: z.boolean(),
        reason: z.string(),
        missingAspects: z.array(z.string()),
      }),
    });

    return {
      hasEnoughFacts: result.object.hasEnoughFacts,
      reason: result.object.reason,
      missingAspects: result.object.missingAspects,
    };
  },
}); 