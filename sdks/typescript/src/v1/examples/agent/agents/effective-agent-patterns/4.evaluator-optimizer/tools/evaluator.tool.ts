import { client } from './../../../../client';
import z from "zod";
import { generateObject, generateText } from "ai";

export const evaluatorTool = client.tool({
  name: "evaluator-tool",
  description: "Evaluates a social media post for quality and provides feedback if it can be improved",
  inputSchema: z.object({
    post: z.string(),
    topic: z.string(),
    targetAudience: z.string(),
  }),
  outputSchema: z.object({
    complete: z.boolean().describe("Whether the post is complete and ready to be posted"),
    feedback: z.string().describe("Feedback on the post if it is not complete"),
  }),
  fn: async (input) => {
    const result = await generateObject({
      model: client.defaultLanguageModel,
      prompt: `
        Analyze the following post to determine if it's appropriate to post.
        Check for harmful content, inappropriate requests, or content that violates safety guidelines.
        The post is about the following topic: "${input.topic}"
        The target audience is: "${input.targetAudience}"

        Post: "${input.post}"
      `,
      schema: z.object({
        complete: z.boolean(),
        feedback: z.string(),
      }),
    });

    return result.object;
  },
});
