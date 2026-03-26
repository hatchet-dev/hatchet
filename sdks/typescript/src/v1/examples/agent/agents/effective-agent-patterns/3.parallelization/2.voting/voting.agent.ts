import { client } from './../../../../client';
import z from 'zod';
import { safetyVoterTool } from './tools/safety-voter.tool';
import { helpfulnessVoterTool } from './tools/helpfulness-voter.tool';
import { accuracyVoterTool } from './tools/accuracy-voter.tool';

/**
 * VOTING PARALLELIZATION PATTERN
 *
 * Based on Anthropic's "Building Effective Agents" blog post:
 * https://www.anthropic.com/engineering/building-effective-agents
 *
 * Pattern Description:
 * Running the same or similar tasks multiple times to get diverse outputs,
 * then using voting logic to determine the final result. This is the second
 * of two parallelization variations (the other being sectioning).
 *
 * When to use:
 * - Need multiple perspectives for higher confidence results
 * - Quality assurance through consensus
 * - Balancing false positives/negatives with vote thresholds
 *
 * Examples:
 * - Code vulnerability review with multiple specialized prompts
 * - Content moderation with different evaluation criteria
 * - Quality assessment requiring consensus
 *
 * Key Insight:
 * Multiple attempts or perspectives can significantly improve confidence in results,
 * especially for subjective or complex evaluation tasks.
 */

const VotingAgentInput = z.object({
  message: z.string(),
  response: z.string(),
});

const VotingAgentOutput = z.object({
  approved: z.boolean(),
  finalResponse: z.string(),
  votingSummary: z.string(),
});

export const votingAgent = client.agent({
  name: 'voting-agent',
  executionTimeout: '1m',
  inputSchema: VotingAgentInput,
  outputSchema: VotingAgentOutput,
  description: 'Demonstrates voting: multiple parallel evaluations with consensus decision-making',
  fn: async (input, ctx) => {
    // PARALLEL VOTING: Run multiple specialized evaluators simultaneously
    // Each voter focuses on a different aspect of quality evaluation:
    // - Safety: Checks for harmful or inappropriate content
    // - Helpfulness: Evaluates whether the response actually helps the user
    // - Accuracy: Assesses factual correctness and reliability
    //
    // This follows Anthropic's pattern of using multiple specialized evaluators
    // rather than trying to do all evaluation in a single call
    const [safetyVote, helpfulnessVote, accuracyVote] = await Promise.all([
      safetyVoterTool.run({
        message: input.message,
        response: input.response,
      }),
      helpfulnessVoterTool.run({
        message: input.message,
        response: input.response,
      }),
      accuracyVoterTool.run({
        message: input.message,
        response: input.response,
      }),
    ]);

    // VOTE COUNTING: Aggregate the individual votes
    const votes = [safetyVote.approve, helpfulnessVote.approve, accuracyVote.approve];
    const approvalCount = votes.filter((vote) => vote).length;
    const totalVotes = votes.length;

    // CONSENSUS DECISION: Require majority approval
    // This threshold can be adjusted based on your needs:
    // - Higher threshold (e.g., unanimous) for more conservative decisions
    // - Lower threshold for more permissive decisions
    // - Different thresholds for different types of content
    const approved = approvalCount >= Math.ceil(totalVotes / 2);

    // TRANSPARENCY: Create detailed voting summary
    // This provides transparency into the decision-making process,
    // which is crucial for debugging and building trust
    const votingSummary = `Voting Results (${approvalCount}/${totalVotes} approved):
- Safety: ${safetyVote.approve ? '✓' : '✗'} - ${safetyVote.reason}
- Helpfulness: ${helpfulnessVote.approve ? '✓' : '✗'} - ${helpfulnessVote.reason}
- Accuracy: ${accuracyVote.approve ? '✓' : '✗'} - ${accuracyVote.reason}`;

    return {
      approved,
      finalResponse: approved
        ? input.response
        : 'I apologize, but I cannot provide that response as it did not meet our quality and safety standards.',
      votingSummary,
    };
  },
});
