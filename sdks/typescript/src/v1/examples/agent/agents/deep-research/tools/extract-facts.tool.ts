import { z } from "zod";
import { generateObject } from "ai";
import { openai } from "@ai-sdk/openai";
import { client } from './../../../client';

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

const FactSchema = z.object({
  text: z.string(),
  sourceIndex: z.number(),
});

const ExtractFactsOutputSchema = z.object({
  facts: z.array(FactSchema),
});

export const extractFacts = client.tool({
  name: "extract-facts",
  description: "Extract relevant facts from a source that are related to a query",
  inputSchema: ExtractFactsInputSchema,
  outputSchema: ExtractFactsOutputSchema,
  fn: async (input, ctx) => {
    const result = await generateObject({
      abortSignal: ctx.abortController.signal,
      prompt: `
Extract relevant facts from the following source that are related to this query:
"""${input.query}"""

Source:
"""${input.source}"""

Extract only factual statements that are directly relevant to the query. Each fact should be a complete, standalone statement.
`,
      model: openai("gpt-4.1-mini"),
      schema: z.object({
        facts: z.array(z.string()),
      }),
    });

    return {
      facts: result.object.facts.map((fact) => ({
        text: fact,
        sourceIndex: input.sourceInfo.index,
      })),
    };
  },
});
