import { z } from 'zod';
import { generateText, zodSchema } from 'ai';
import { TaskWorkflowDeclaration } from '../declaration';
import { HatchetClient, Registerable } from './client';

export interface ToolDeclaration<
  InputSchema extends z.ZodType,
  OutputSchema extends z.ZodType,
> extends TaskWorkflowDeclaration<z.infer<InputSchema>, z.infer<OutputSchema>> {
  inputSchema: InputSchema;
  outputSchema: OutputSchema;
  description: string;
}

// Helper to pull the right ToolDeclaration from T by its `name`
type ToolByName<
  T extends readonly ToolDeclaration<any, any>[],
  N extends T[number]['name'],
> = Extract<T[number], { name: N }>;

// Map over each name and give it the correct input/output
type TransformersFor<T extends readonly ToolDeclaration<any, any>[]> = {
  [N in T[number]['name']]: (
    output: z.infer<ToolByName<T, N>['outputSchema']>,
    args: z.infer<ToolByName<T, N>['inputSchema']>
  ) => Promise<any>;
};

type ToolResultMap<T extends readonly ToolDeclaration<any, any>[]> = {
  [N in T[number]['name']]: {
    name: N;
    output: z.infer<Extract<T[number], { name: N }>['outputSchema']>;
    args: z.infer<Extract<T[number], { name: N }>['inputSchema']>;
  };
}[T[number]['name']];

/**
 * Options used when creating a `Toolbox` instance.
 *
 * @typeParam T - Immutable array of `ToolDeclaration`s that will populate the toolbox.
 */
export interface CreateToolboxOpts<T extends ReadonlyArray<ToolDeclaration<any, any>>> {
  /** Array of tool declarations to register on the toolbox. */
  tools: T;
}

export type ToolSet = {
  [key: string]: {
    /**
     The schema of the input that the tool expects. The language model will use this to generate the input.
     It is also used to validate the output of the language model.
     Use descriptions to make the input understandable for the language model.
     */
    parameters: any;
    /**
     An optional description of what the tool does.
     Will be used by the language model to decide whether to use the tool.
     Not used for provider-defined tools.
     */
    description?: string;
  };
};

/**
 * Input for the pick workflow.
 */
type PickInput = {
  /**
   * The prompt to use for the pick workflow.
   */
  prompt: string;

  /**
   * The maximum number of tools to allow the pick workflow to take.
   */
  maxTools?: number;
};

/**
 * Runtime helper that exposes a collection of Hatchet workflows as "tools" which can be
 * automatically selected and executed by a language model using the OpenAI function-calling interface.
 *
 * The class wires the supplied tool declarations into the Hatchet runtime and also
 * registers an internal `pick-tool` workflow that decides—at execution time—what tool(s)
 * should be invoked to satisfy a natural-language prompt.
 *
 * @typeParam T - Immutable array of `ToolDeclaration`s that are made available in this toolbox.
 */
export class Toolbox<T extends ReadonlyArray<ToolDeclaration<any, any>>> implements Registerable {
  private toolboxKey: string;
  toolSetForAI: ToolSet;

  /**
   * Creates a new `Toolbox`.
   *
   * @param props  - Toolbox construction options containing the tool declarations.
   * @param client - The `HatchetClient` client used to register and execute workflows.
   */
  constructor(
    private props: CreateToolboxOpts<T>,
    private client: HatchetClient
  ) {
    // Generate a key for this toolbox based on tool names
    this.toolboxKey = Array.from(this.props.tools)
      .map((t) => t.name)
      .sort()
      .join(':');

    // Create toolset for AI SDK using the actual Zod schemas
    this.toolSetForAI = Array.from(this.props.tools).reduce<ToolSet>((acc, tool) => {
      if (!tool.name || !tool.inputSchema) {
        throw new Error(`Tool must have a name and inputSchema`);
      }

      return {
        ...acc,
        [tool.name]: {
          parameters: zodSchema(tool.inputSchema),
          description: tool.description,
        },
      };
    }, {});
  }

  /**
   * Implements the `Registerable` interface so the toolbox and its supporting workflows
   * can be registered with Hatchet.
   *
   * @returns All user-supplied tool declarations plus the internal `pick-tool` workflow.
   */
  get register(): TaskWorkflowDeclaration<any, any>[] {
    return [...this.props.tools, pickToolFactory(this.client)];
  }

  /**
   * Uses the language model to choose up to `maxTools` tools from this toolbox that best satisfy
   * the provided prompt. Only the selection step is performed—no tool execution happens here.
   *
   * @param prompt   - Natural-language description of what the caller wants to achieve.
   * @param [maxTools] - Optional upper bound on how many tools may be selected (defaults to 1).
   *
   * @returns An array containing the chosen tool names together with the generated input arguments.
   */
  async pick({ prompt, maxTools }: PickInput) {
    const result = await pickToolFactory(this.client).run({
      prompt,
      toolboxKey: this.toolboxKey,
      maxTools,
    });

    return result.steps.flatMap((step) =>
      step.map((toolCall) => ({
        name: toolCall.toolName,
        input: toolCall.args,
      }))
    );
  }

  /**
   * Convenience method that first runs `pick` and then immediately executes the chosen tool(s).
   *
   * When `maxTools` is omitted or set to `1` the return value is a single `ToolResultMap` object.
   * For values greater than `1` an array of `ToolResultMap` objects is returned in the order the
   * tools were selected.
   *
   * @typeParam R - Inferred map of tool names to transformer functions derived from the toolbox.
   * @param opts - Same input accepted by `pick`; the `maxTools` property determines the shape of the response.
   */
  async pickAndRun<R extends TransformersFor<T>>(
    opts: Omit<PickInput, 'maxTools'> & { maxTools?: undefined }
  ): Promise<ToolResultMap<T>>;

  async pickAndRun<R extends TransformersFor<T>>(
    opts: PickInput & { maxTools: 1 }
  ): Promise<ToolResultMap<T>>;

  // Overload: `maxTools` > 1  ➜ array of results
  async pickAndRun<R extends TransformersFor<T>>(
    opts: PickInput & { maxTools: number }
  ): Promise<ToolResultMap<T>[]>;

  // Implementation
  async pickAndRun<R extends TransformersFor<T>>(opts: PickInput): Promise<any> {
    // 1) pick tools
    const picked = await this.pick(opts);

    // 2) run them
    const results = await Promise.all(
      picked.map(async ({ name, input }) => {
        const tool = this.props.tools.find((t) => t.name === name);
        if (!tool) {
          throw new Error(`Tool "${name}" not found in toolbox`);
        }
        return await tool.run(input);
      })
    );

    // 3) zip back into the correctly typed union
    const zipped = picked.map(({ name, input }, i) => ({
      name,
      output: results[i],
      args: input,
    })) as ToolResultMap<T>[];

    // 4) Return single object or array based on opts.maxTools
    if (opts.maxTools === undefined || opts.maxTools === 1) {
      return zipped[0];
    }

    return zipped;
  }

  /**
   * Helper method to assert that the toolbox result is exhaustive
   * for handling results in a switch statement.
   *
   * @param result - The result to assert is exhaustive.
   * @returns The result.
   * @throws An error if the result is not exhaustive.
   */
  assertExhaustive(result: never): never {
    throw new Error(`Unhandled toolbox result: ${(result as any).name}`);
  }

  /**
   * Gets the original tool declarations (used internally by pick-tool)
   */
  getTools(): T {
    return this.props.tools;
  }
}

type PickInputWithToolboxKey = PickInput & {
  toolboxKey: string;
};

const pickToolFactory = (icepick: HatchetClient) =>
  icepick.task({
    name: 'pick-tool',
    executionTimeout: '5m',
    fn: async (input: PickInputWithToolboxKey) => {
      // Get the toolbox from the client using the key
      const toolbox = icepick._getToolbox(input.toolboxKey);
      if (!toolbox) {
        throw new Error(`Toolbox not found for key: ${input.toolboxKey}`);
      }

      // Use the toolbox's AI-ready toolset
      const { steps } = await generateText({
        model: icepick.defaultLanguageModel!,
        tools: toolbox.toolSetForAI,
        maxSteps: input.maxTools ?? 1,
        prompt: input.prompt,
      });

      return { steps: steps.map((step) => step.toolCalls) };
    },
  });
