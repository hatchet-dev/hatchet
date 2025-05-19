import { hatchet } from '@/hatchet.client';
import { z } from 'zod';
import { generateObject } from 'ai';
import { openai } from '@ai-sdk/openai';

const ExtractFactsInputSchema = z.object({
  source: z.string(),
  query: z.string(),
  sourceInfo: z.object({
    url: z.string(),
    title: z.string().optional(),
    index: z.number(),
  }),
});

type ExtractFactsInput = z.infer<typeof ExtractFactsInputSchema>;

type Fact = {
  text: string;
  sourceIndex: number;
};

type ExtractFactsOutput = {
  facts: Fact[];
};

export const extractFacts = hatchet.task({
  name: 'extract-facts',
  fn: async (input: ExtractFactsInput): Promise<ExtractFactsOutput> => {
    const validatedInput = ExtractFactsInputSchema.parse(input);

    const result = await generateObject({
      prompt: `
Extract relevant facts from the following source that are related to this query:
"""${validatedInput.query}"""

Source:
"""${validatedInput.source}"""

Extract only factual statements that are directly relevant to the query. Each fact should be a complete, standalone statement.
`,
      model: openai('gpt-4.1-mini'),
      schema: z.object({
        facts: z.array(z.string()),
      }),
    });

    return {
      facts: result.object.facts.map(fact => ({
        text: fact,
        sourceIndex: validatedInput.sourceInfo.index,
      })),
    };
  },
}); 