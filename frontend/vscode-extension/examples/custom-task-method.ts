/**
 * Example: custom wrapper with non-default task method and parent property names.
 *
 * When your wrapper uses a method name other than `task`, or a dependency
 * property name other than `parents`, declare them with:
 *   @hatchet-task-method  <methodName>
 *   @hatchet-task-parents <propName>
 */
import Hatchet, { type WorkflowDeclaration } from '@hatchet-dev/typescript-sdk';

type JsonObject = Record<string, unknown>;

interface StepOptions<TInput extends JsonObject, TOutput extends JsonObject> {
  fn: (input: TInput) => Promise<TOutput>;
  /** Steps that must complete before this one. */
  dependsOn?: StepHandle<JsonObject>[];
}

export interface StepHandle<TOutput extends JsonObject> {
  readonly name: string;
  readonly _def: ReturnType<WorkflowDeclaration['task']>;
}

export interface PipelineBuilder<TInput extends JsonObject> {
  /** Register a pipeline step. */
  step<TOutput extends JsonObject>(
    name: string,
    options: StepOptions<TInput, TOutput>,
  ): StepHandle<TOutput>;
  build(): WorkflowDeclaration;
}

const hatchet = Hatchet.init();

/**
 * Create a pipeline builder where steps are registered with `.step()` and
 * dependencies are expressed via `dependsOn`.
 *
 * @hatchet-workflow
 * @hatchet-task-method step
 * @hatchet-task-parents dependsOn
 */
export function createPipelineBuilder<TInput extends JsonObject>(options: {
  name: string;
}): PipelineBuilder<TInput> {
  const wf = hatchet.workflow<TInput>({ name: options.name });

  return {
    step<TOutput extends JsonObject>(
      name: string,
      stepOpts: StepOptions<TInput, TOutput>,
    ): StepHandle<TOutput> {
      const def = wf.task({
        name,
        parents: stepOpts.dependsOn?.map((s) => s._def),
        fn: async (input: TInput) => stepOpts.fn(input),
      });
      return { name, _def: def };
    },
    build(): WorkflowDeclaration {
      return wf;
    },
  };
}

// ── Usage ─────────────────────────────────────────────────────────────────────
// The extension detects `imagePipeline` via the @hatchet-workflow annotation
// on createPipelineBuilder, using `step` as the task method and `dependsOn`
// as the parents property.

const imagePipeline = createPipelineBuilder<{ imageUrl: string }>({
  name: 'image-processing-pipeline',
});

const download = imagePipeline.step('download', {
  fn: async (input) => ({ bytes: new Uint8Array() }),
});

const resize = imagePipeline.step('resize', {
  dependsOn: [download],
  fn: async (input) => ({ resized: true }),
});

const compress = imagePipeline.step('compress', {
  dependsOn: [resize],
  fn: async (input) => ({ compressed: true }),
});

const upload = imagePipeline.step('upload', {
  dependsOn: [compress],
  fn: async (input) => ({ url: '' }),
});

export default imagePipeline.build();
