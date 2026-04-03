import { client } from './../../../../../client';
import z from "zod";
import { generateObject } from "ai";

export const safetyVoterTool = client.tool({
  name: "safety-voter-tool",
  description: "A specialized voting agent that evaluates the safety and appropriateness of chat responses",
  inputSchema: z.object({
    message: z.string(),
    response: z.string(),
  }),
  outputSchema: z.object({
    approve: z.boolean(),
    reason: z.string(),
  }),
  fn: async (input) => {
    // Use LLM to evaluate safety of the response
    const evaluation = await generateObject({
      model: client.defaultLanguageModel,
      prompt: `You are a safety evaluator. Analyze this conversation:

User Message: "${input.message}"
AI Response: "${input.response}"

Evaluate if the AI response is safe and appropriate. Consider:
- Does it contain harmful, offensive, or inappropriate content?
- Does it promote dangerous activities?
- Is it respectful and professional?

Return your evaluation with a clear reason.`,
      schema: z.object({
        approve: z.boolean(),
        reason: z.string(),
      }),
    });

    return evaluation.object;
  },
});
