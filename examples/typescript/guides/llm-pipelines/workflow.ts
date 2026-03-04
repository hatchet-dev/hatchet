import { hatchet } from '../../hatchet-client';
import { generate } from './mock-llm';

type PipelineInput = { prompt: string };

// > Step 01 Define Pipeline
const llmWf = hatchet.workflow<PipelineInput>({ name: 'LLMPipeline' });

const promptTask = llmWf.task({
  name: 'prompt-task',
  fn: async (input) => ({ prompt: input.prompt }),
});


// > Step 02 Prompt Task
function buildPrompt(userInput: string, context = ''): string {
  return `Process the following: ${userInput}${context ? `\nContext: ${context}` : ''}`;
}

// > Step 03 Validate Task
const generateTask = llmWf.task({
  name: 'generate-task',
  parents: [promptTask],
  fn: async (input, ctx) => {
    const prev = await ctx.parentOutput(promptTask);
    const output = generate(prev.prompt);
    if (!output.valid) throw new Error('Validation failed');
    return output;
  },
});


export { llmWf };
