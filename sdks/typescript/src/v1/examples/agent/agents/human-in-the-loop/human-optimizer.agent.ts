import { client } from './../../client';
import z from "zod";
import { generatorTool } from "./tools/generator.tool";
import { sendToSlackTool } from "./tools/send-to-slack.tool";

/**
 * Human-in-the-loop: Generator with Human Feedback
 *
 * Extends the Evaluator-Optimizer pattern to include a human in the loop.
 * Based on ../effective-agent-patterns/4.evaluator-optimizer but replaces
 * LLM evaluation with human evaluation via Slack.
 *
 * Pattern Description:
 * An LLM generates content while a human provides evaluation and feedback
 * in a loop, iteratively improving the output. This combines automated
 * generation with human judgment and expertise.
 *
 * When to use:
 * - Human judgment/expertise is critical for evaluation
 * - Quality standards are subjective or domain-specific
 * - Human feedback can provide nuanced improvements
 * - Iterative refinement with human oversight adds value
 * - Real-time human approval is required
 *
 * Examples:
 * - Content creation requiring brand voice approval
 * - Marketing copy needing stakeholder sign-off
 * - Creative writing with editorial feedback
 * - Technical documentation requiring expert review
 * - Social media posts needing compliance approval
 *
 * Key Insight:
 * This pattern works well when human expertise and judgment are irreplaceable
 * for evaluation, while still leveraging LLM efficiency for generation and iteration.
 */

const EvaluatorOptimizerAgentInput = z.object({
  topic: z.string(),
  targetAudience: z.string(),
});

const EvaluatorOptimizerAgentOutput = z.object({
  post: z.string(),
  iterations: z.number(),
});

type FeedbackEvent = {
  messageId: string;
  approved: boolean;
  feedback?: string;
}

export const evaluatorOptimizerAgent = client.agent({
  name: "human-optimizer-agent",
  executionTimeout: "2m",
  inputSchema: EvaluatorOptimizerAgentInput,
  outputSchema: EvaluatorOptimizerAgentOutput,
  description: "Demonstrates human-in-the-loop: iterative generation with human feedback via Slack",
  fn: async (input, ctx) => {

    let post: string | undefined;
    let feedback: string | undefined;
    let iterations = 0;

    // ITERATIVE IMPROVEMENT LOOP
    // The loop continues until either:
    // 1. The human approves the output via Slack
    // 2. We reach the maximum number of iterations (prevents infinite loops)
    for (let i = 0; i < 3; i++) {
      iterations++;
      // GENERATION STEP: Create or improve the content
      // The generator takes into account:
      // - Original requirements (topic, target audience)
      // - Previous attempt (if any)
      // - Feedback from human reviewer (if any)
      const { post: newPost } = await generatorTool.run({
        topic: input.topic,
        targetAudience: input.targetAudience,
        previousPost: post,
        previousFeedback: feedback
      });
      post = newPost;

      // HUMAN REVIEW STEP: Send content to Slack for human evaluation
      // This sends the generated post to Slack where a human can:
      // - Approve the content (if satisfactory)
      // - Provide specific feedback for improvement (if changes needed)

      const slackMessage = await sendToSlackTool.run({
        post: post,
        topic: input.topic,
        targetAudience: input.targetAudience
      });

      // dispatch an event on an approve or reject with feedback button
      // icepick.events.push<FeedbackEvent>("feedback:create", {
      //   messageId: slackResult.messageId,
      //   approved: false,
      // })

      // Wait for human feedback via Slack interaction
      const feedbackEvent = await ctx.waitFor({
          eventKey: "feedback:create",
          expression: `input.messageId == "${slackMessage.messageId}"`,
      })

      const event = feedbackEvent["feedback:create"] as FeedbackEvent

      // COMPLETION CHECK: If human approves, return the result
      if (event.approved) {
        return {
          post: post,
          iterations: iterations,
        };
      }

      feedback = event.feedback;

      // If not approved, the loop continues with the human feedback for the next iteration
    }

    // FALLBACK: If we've reached max iterations without human approval
    // This prevents infinite loops while still returning the best attempt
    if (!post) throw new Error("I was unable to generate a post");

    return {
      post: post,
      iterations: iterations,
    };
  },
});
