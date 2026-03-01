import { hatchet } from '../../hatchet-client';
import { mockGenerate, mockEvaluate } from './mock-llm';

type GeneratorInput = {
  topic: string;
  audience: string;
  previousDraft?: string;
  feedback?: string;
};

type EvaluatorInput = {
  draft: string;
  topic: string;
  audience: string;
};

// > Step 01 Define Tasks
const generatorTask = hatchet.task({
  name: 'generate-draft',
  fn: async (input: GeneratorInput) => {
    const prompt = input.feedback
      ? `Improve this draft.\n\nDraft: ${input.previousDraft}\nFeedback: ${input.feedback}`
      : `Write a social media post about "${input.topic}" for ${input.audience}. Under 100 words.`;
    return { draft: mockGenerate(prompt) };
  },
});

const evaluatorTask = hatchet.task({
  name: 'evaluate-draft',
  fn: async (input: EvaluatorInput) => {
    return mockEvaluate(input.draft);
  },
});
// !!

// > Step 02 Optimization Loop
const optimizerTask = hatchet.durableTask({
  name: 'evaluator-optimizer',
  executionTimeout: '5m',
  fn: async (input: { topic: string; audience: string }) => {
    const maxIterations = 3;
    const threshold = 0.8;
    let draft = '';
    let feedback = '';

    for (let i = 0; i < maxIterations; i++) {
      const generated = await generatorTask.run({
        topic: input.topic,
        audience: input.audience,
        previousDraft: draft || undefined,
        feedback: feedback || undefined,
      });
      draft = generated.draft;

      const evaluation = await evaluatorTask.run({
        draft,
        topic: input.topic,
        audience: input.audience,
      });

      if (evaluation.score >= threshold) {
        return { draft, iterations: i + 1, score: evaluation.score };
      }
      feedback = evaluation.feedback;
    }

    return { draft, iterations: maxIterations, score: -1 };
  },
});
// !!

export { generatorTask, evaluatorTask, optimizerTask };
