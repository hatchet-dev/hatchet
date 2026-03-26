import { z } from "zod";
import { generateObject } from "ai";
import { openai } from "@ai-sdk/openai";
import { client } from './../../../client';

const JudgeFactsInputSchema = z.object({
  query: z.string(),
  facts: z.array(z.string()),
});

const JudgeFactsOutputSchema = z.object({
  hasEnoughFacts: z.boolean(),
  reason: z.string(),
  missingAspects: z.array(z.string()),
});

export type JudgeFactsOutput = z.infer<typeof JudgeFactsOutputSchema>;

export const judgeFacts = client.tool({
  name: "judge-facts",
  description: "Judge if we have enough facts to comprehensively answer a query",
  inputSchema: JudgeFactsInputSchema,
  outputSchema: JudgeFactsOutputSchema,
  fn: async (input, ctx) => {
    const result = await generateObject({
      abortSignal: ctx.abortController.signal,
      prompt: `
Evaluate if we have enough facts to comprehensively answer this query:
"""${input.query}"""

Current facts:
${input.facts.map((fact, i) => `${i + 1}. ${fact}`).join("\n")}

Consider:
1. Are there any key aspects of the query that aren't covered by the current facts?
2. Are the facts diverse enough to provide a complete picture?
3. Are there any gaps in the information that would prevent a comprehensive answer?
4. Are there any technical jargon words that are not defined in the facts that require additional research?
`,
      model: openai("gpt-4.1-mini"),
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
