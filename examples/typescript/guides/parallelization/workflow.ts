import { hatchet } from '../../hatchet-client';
import { mockGenerateContent, mockSafetyCheck, mockEvaluate } from './mock-llm';

type MessageInput = { message: string };

// > Step 01 Parallel Tasks
const contentTask = hatchet.task({
  name: 'generate-content',
  fn: async (input: MessageInput) => {
    return { content: mockGenerateContent(input.message) };
  },
});

const safetyTask = hatchet.task({
  name: 'safety-check',
  fn: async (input: MessageInput) => {
    return mockSafetyCheck(input.message);
  },
});

const evaluateTask = hatchet.task({
  name: 'evaluate-content',
  fn: async (input: { content: string }) => {
    return mockEvaluate(input.content);
  },
});

// > Step 02 Sectioning
const sectioningTask = hatchet.durableTask({
  name: 'parallel-sectioning',
  executionTimeout: '2m',
  fn: async (input: MessageInput) => {
    const [content, safety] = await Promise.all([
      contentTask.run(input),
      safetyTask.run(input),
    ]);

    if (!safety.safe) {
      return { blocked: true, reason: safety.reason };
    }
    return { blocked: false, content: content.content };
  },
});

// > Step 03 Voting
const votingTask = hatchet.durableTask({
  name: 'parallel-voting',
  executionTimeout: '3m',
  fn: async (input: { content: string }) => {
    const votes = await Promise.all([
      evaluateTask.run(input),
      evaluateTask.run(input),
      evaluateTask.run(input),
    ]);

    const approvals = votes.filter((v) => v.approved).length;
    const avgScore = votes.reduce((sum, v) => sum + v.score, 0) / votes.length;

    return {
      approved: approvals >= 2,
      averageScore: avgScore,
      votes: votes.length,
    };
  },
});

export { contentTask, safetyTask, evaluateTask, sectioningTask, votingTask };
