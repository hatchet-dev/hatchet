import { client } from './../../../client';
import z from "zod";
import { generateText } from "ai";

export const generatorTool = client.tool({
  name: "generator-tool",
  description: "Generates a social media post",
  inputSchema: z.object({
    topic: z.string(),
    targetAudience: z.string(),
    previousPost: z.string().optional(),
    previousFeedback: z.string().optional(),
  }),
  outputSchema: z.object({
    post: z.string(),
  }),
  fn: async (input) => {
    const result = await generateText({
      model: client.defaultLanguageModel,
      prompt: `
        Generate a social media post for the following topic.
        This should be the detailed, substantive part of the response that directly addresses the user's query.
        Provide helpful information, explanations, or answers as appropriate.
        The post should be 100 words or less.

        Topic: "${input.topic}"
        Target Audience: "${input.targetAudience}"

        ${input.previousFeedback ? `Improve the post based on the following feedback: "${input.previousFeedback}"` : ""}
        ${input.previousPost ? `Previous Post: "${input.previousPost}"` : ""}

        Provide only the post text, no additional formatting or labels.
      `,
    });

    return {
      post: result.text.trim(),
    };
  },
});
