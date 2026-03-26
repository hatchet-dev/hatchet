import { client } from './../../../../client';
import z from "zod";
import { appropriatenessCheckTool } from "./tools/appropriateness.tool";
import { mainContentTool } from "./tools/main-content.tool";

/**
 * SECTIONING PARALLELIZATION PATTERN
 *
 * Based on Anthropic's "Building Effective Agents" blog post:
 * https://www.anthropic.com/engineering/building-effective-agents
 *
 * Pattern Description:
 * Breaking a task into independent subtasks that run simultaneously, then
 * aggregating results programmatically. This is one of two parallelization
 * variations (the other being voting).
 *
 * When to use:
 * - Independent subtasks can be parallelized for speed
 * - Multiple considerations need separate focused attention
 * - Implementing guardrails alongside main processing
 *
 * Examples:
 * - Guardrails: One model processes queries while another screens for inappropriate content
 * - Code review: Multiple aspects evaluated simultaneously
 * - Multi-faceted analysis requiring separate specialized attention
 *
 * Key Insight:
 * Anthropic found that LLMs generally perform better when each consideration
 * is handled by a separate LLM call, allowing focused attention on each specific aspect.
 */

const SectioningAgentInput = z.object({
  message: z.string(),
});

const SectioningAgentOutput = z.object({
  response: z.string(),
  isAppropriate: z.boolean(),
});

export const sectioningAgent = client.agent({
  name: "sectioning-agent",
  executionTimeout: "2m",
  inputSchema: SectioningAgentInput,
  outputSchema: SectioningAgentOutput,
  description: "Demonstrates sectioning: parallel independent subtasks with focused attention",
  fn: async (input, ctx) => {

    // PARALLEL EXECUTION: Run independent subtasks simultaneously
    // This is the core of sectioning - instead of running tasks sequentially,
    // we run them in parallel because they address different concerns
    //
    // Task 1: Appropriateness check (guardrail)
    // Task 2: Main content generation
    //
    // These are independent - the appropriateness check doesn't need the main content
    // to do its job, and vice versa. This allows for significant speed improvements.
    const [{isAppropriate, reason}, mainResult] = await Promise.all([
      appropriatenessCheckTool.run({ message: input.message }),
      mainContentTool.run({ message: input.message }),
    ]);

    // AGGREGATION: Combine results with business logic
    // The appropriateness check acts as a guardrail - if content is inappropriate,
    // we discard the main content and return a safety message
    if (!isAppropriate) {
        return {
          response: `I cannot provide a response to that request. ${reason}`,
          isAppropriate: false,
        };
      }

    // If appropriate, return the main content
    return {
      response: mainResult.mainContent,
      isAppropriate: true,
    };
  },
});

 [sectioningAgent];
