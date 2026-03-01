import { hatchet } from '../../hatchet-client';
import { mockOrchestratorLlm, mockSpecialistLlm } from './mock-llm';

type SpecialistInput = { task: string; context?: string };

// > Step 01 Specialist Agents
const researchTask = hatchet.task({
  name: 'research-specialist',
  executionTimeout: '3m',
  fn: async (input: SpecialistInput) => {
    return { result: mockSpecialistLlm(input.task, 'research') };
  },
});

const writingTask = hatchet.task({
  name: 'writing-specialist',
  executionTimeout: '2m',
  fn: async (input: SpecialistInput) => {
    return { result: mockSpecialistLlm(input.task, 'writing') };
  },
});

const codeTask = hatchet.task({
  name: 'code-specialist',
  executionTimeout: '2m',
  fn: async (input: SpecialistInput) => {
    return { result: mockSpecialistLlm(input.task, 'code') };
  },
});

// > Step 02 Orchestrator Loop
const specialists: Record<string, typeof researchTask> = {
  research: researchTask,
  writing: writingTask,
  code: codeTask,
};

const orchestrator = hatchet.durableTask({
  name: 'multi-agent-orchestrator',
  executionTimeout: '15m',
  fn: async (input: { goal: string }) => {
    const messages: Array<{ role: string; content: string }> = [
      { role: 'user', content: input.goal },
    ];

    for (let i = 0; i < 10; i++) {
      const response = mockOrchestratorLlm(messages);

      if (response.done) return { result: response.content };

      const specialist = specialists[response.toolCall!.name];
      if (!specialist) throw new Error(`Unknown specialist: ${response.toolCall!.name}`);

      const { result } = await specialist.run({
        task: response.toolCall!.args.task,
        context: messages.map((m) => m.content).join('\n'),
      });

      messages.push(
        { role: 'assistant', content: `Called ${response.toolCall!.name}` },
        { role: 'tool', content: result }
      );
    }

    return { result: 'Max iterations reached' };
  },
});

export { researchTask, writingTask, codeTask, orchestrator };
