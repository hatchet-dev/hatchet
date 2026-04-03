import { client } from './../../../../../client';
import z from "zod";
import { generateObject } from "ai";

export const accuracyVoterTool = icepick.tool({
  name: "accuracy-voter-tool",
  description: "A specialized voting agent that evaluates the accuracy and reasoning quality of chat responses",
  inputSchema: z.object({
    message: z.string(),
    response: z.string(),
  }),
  outputSchema: z.object({
    approve: z.boolean(),
    reason: z.string(),
  }),
  fn: async (input) => {
    // Use LLM to evaluate accuracy of the response
    const evaluation = await generateObject({
      model: client.defaultLanguageModel,
      prompt: `You are an accuracy evaluator. Analyze this conversation:

User Message: "${input.message}"
AI Response: "${input.response}"

Evaluate if the AI response is accurate and well-reasoned. Consider:
- Are the facts presented correct to the best of your knowledge?
- Is the reasoning sound and logical?
- Does it avoid making unsubstantiated claims?
- Is it appropriately cautious about uncertain information?

Return your evaluation with a clear reason.`,
      schema: z.object({
        approve: z.boolean(),
        reason: z.string(),
      }),
    });

    return evaluation.object;
  },
});
