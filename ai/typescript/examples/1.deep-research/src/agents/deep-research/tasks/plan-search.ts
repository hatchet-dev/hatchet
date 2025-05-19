import { hatchet } from '@/hatchet.client';
import { z } from 'zod';
import { generateObject } from 'ai';
import { openai } from '@ai-sdk/openai';

export const PlanSearchInputSchema = z.object({
  query: z.string(),
  existingFacts: z.array(z.string()).optional(),
  missingAspects: z.array(z.string()).optional(),
});

export type PlanSearchInput = z.infer<typeof PlanSearchInputSchema>;

export type PlanSearchOutput = {
  queries: string[];
  reasoning: string;
};

export const planSearch = hatchet.task({
  name: 'plan-search',
  fn: async (input: PlanSearchInput): Promise<PlanSearchOutput> => {
    const validatedInput = PlanSearchInputSchema.parse(input);

    const result = await generateObject({
      prompt: `
Plan search queries to find information about this topic:
"""${validatedInput.query}"""

${validatedInput.existingFacts ? `
We already have these facts:
${validatedInput.existingFacts.map((fact, i) => `${i + 1}. ${fact}`).join('\n')}
` : ''}

${validatedInput.missingAspects ? `
We need to find information about these missing aspects:
${validatedInput.missingAspects.map((aspect, i) => `${i + 1}. ${aspect}`).join('\n')}
` : ''}

Generate 3-5 specific search queries that will help us find new, relevant information.
The queries should:
1. Focus on finding information about missing aspects
2. Avoid duplicating information we already have
3. Be specific enough to find relevant sources
4. Use different angles or perspectives to ensure diverse information
`,
      model: openai('gpt-4.1-mini'),
      schema: z.object({
        queries: z.array(z.string()),
        reasoning: z.string(),
      }),
    });

    return {
      queries: result.object.queries,
      reasoning: result.object.reasoning,
    };
  },
});
