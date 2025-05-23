import { BaseWorkflowDeclaration } from "@hatchet-dev/typescript-sdk/v1/declaration";
import { z } from "zod";
import { generateText } from "ai";
import { openai } from "@ai-sdk/openai";
import { hatchet } from "@/hatchet.client";
import { zodSchema } from 'ai';

export interface TaskboxProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  tasks: {task: BaseWorkflowDeclaration<any, any>, schema: z.ZodType}[];
}

export type ToolSet = {
    [key: string]: {
        /**
  The schema of the input that the tool expects. The language model will use this to generate the input.
  It is also used to validate the output of the language model.
  Use descriptions to make the input understandable for the language model.
     */
  parameters: z.ZodType;
  /**
An optional description of what the tool does.
Will be used by the language model to decide whether to use the tool.
Not used for provider-defined tools.
   */
  description?: string;
    }
}

type SerializedToolSet = {
  [key: string]: {
    typeName: string;
    description: string;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    parameters: any // TODO ReturnType<typeof zodToJsonSchema>;
  }
}

class Taskbox {

  toolset: SerializedToolSet;

  constructor(private props: TaskboxProps) {
    this.toolset = this.props.tasks.reduce<SerializedToolSet>((acc, {task, schema}) => {
        return {
            ...acc,
            [task.name]: {
                typeName: task.name,
                description: task.definition.description || '',
                parameters: zodSchema(schema)
            }
        }
    }, {});
  }


  
  get register() {
    return [...this.props.tasks.map(({task}) => task), pick];
  }


  async pick(prompt: string) {
    return pick.run({prompt, toolset: this.toolset});
  }

}

type PickInput = {
  prompt: string;
  toolset: SerializedToolSet;
}

const pick = hatchet.task({
  name: 'pick',
  fn: async (input: PickInput) => {

    console.log(JSON.stringify(input.toolset, null, 2));
    const { steps } = await generateText({
        model: openai('gpt-4o-mini'),
        tools: input.toolset,
        maxSteps: 5, // allow up to 5 steps
        prompt: input.prompt,
      });

    return {steps: steps.map(step => step.toolCalls)};
  }
});


export const taskbox = (props: TaskboxProps) => {
  return new Taskbox(props);
};
