import { client } from './../../../client';
import z from 'zod';
import { evaluatorTool } from './tools/evaluator.tool';
import { generatorTool } from './tools/generator.tool';

/**
 * EVALUATOR-OPTIMIZER PATTERN
 *
 * Based on Anthropic's "Building Effective Agents" blog post:
 * https://www.anthropic.com/engineering/building-effective-agents
 *
 * Pattern Description:
 * One LLM generates a response while another provides evaluation and feedback
 * in a loop, iteratively improving the output. This is analogous to the
 * iterative writing process a human writer might go through.
 *
 * When to use:
 * - Clear evaluation criteria exist
 * - Iterative refinement provides measurable value
 * - LLM can provide useful feedback (similar to human feedback improving results)
 * - Quality improvements possible through iteration
 *
 * Examples:
 * - Literary translation with nuance refinement
 * - Complex search requiring multiple rounds of analysis
 * - Content creation with quality improvement loops
 * - Creative writing with iterative polish
 *
 * Key Insight:
 * This pattern works well when LLM responses can be demonstrably improved
 * when a human articulates feedback, and when the LLM can provide such feedback itself.
 */

const EvaluatorOptimizerAgentInput = z.object({
  topic: z.string(),
  targetAudience: z.string(),
});

const EvaluatorOptimizerAgentOutput = z.object({
  post: z.string(),
  iterations: z.number(),
});

export const evaluatorOptimizerAgent = client.agent({
  name: 'evaluator-optimizer-agent',
  executionTimeout: '2m',
  inputSchema: EvaluatorOptimizerAgentInput,
  outputSchema: EvaluatorOptimizerAgentOutput,
  description: 'Demonstrates evaluator-optimizer: iterative generation and refinement loop',
  fn: async (input, ctx) => {
    let post: string | undefined;
    let feedback: string | undefined;
    let iterations = 0;

    // ITERATIVE IMPROVEMENT LOOP
    // The loop continues until either:
    // 1. The evaluator determines the output is satisfactory (complete = true)
    // 2. We reach the maximum number of iterations (prevents infinite loops)
    for (let i = 0; i < 3; i++) {
      iterations++;
      // GENERATION STEP: Create or improve the content
      // The generator takes into account:
      // - Original requirements (topic, target audience)
      // - Previous attempt (if any)
      // - Feedback from evaluator (if any)
      const { post: newPost } = await generatorTool.run({
        topic: input.topic,
        targetAudience: input.targetAudience,
        previousPost: post,
        previousFeedback: feedback,
      });
      post = newPost;

      // EVALUATION STEP: Assess the generated content
      // The evaluator provides:
      // - A completion flag (is this good enough?)
      // - Specific feedback for improvement (if not complete)
      const evaluatorResult = await evaluatorTool.run({
        post: post,
        topic: input.topic,
        targetAudience: input.targetAudience,
      });

      feedback = evaluatorResult.feedback;

      // COMPLETION CHECK: If evaluator is satisfied, return the result
      if (evaluatorResult.complete) {
        return {
          post: post,
          iterations: iterations,
        };
      }

      // If not complete, the loop continues with the feedback for the next iteration
    }

    // FALLBACK: If we've reached max iterations without completion
    // This prevents infinite loops while still returning the best attempt
    if (!post) throw new Error('I was unable to generate a post');

    return {
      post: post,
      iterations: iterations,
    };
  },
});
