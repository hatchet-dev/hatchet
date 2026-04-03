import { client } from './../../../client';
import z from "zod";

export const sendToSlackTool = client.tool({
  name: "send-to-slack-tool",
  description: "Sends a message to Slack",
  inputSchema: z.object({
    post: z.string(),
    topic: z.string(),
    targetAudience: z.string(),
  }),
  outputSchema: z.object({
    messageId: z.string(),
  }),
  fn: async (input) => {
    // TODO: Implement the tool

    return {
      messageId: "123",
    };
  },
});
