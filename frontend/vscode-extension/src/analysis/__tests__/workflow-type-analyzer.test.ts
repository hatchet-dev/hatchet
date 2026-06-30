import { describe, it, expect, beforeAll } from 'vitest';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { WorkflowTypeAnalyzer, type WorkflowAnchor } from '../workflow-type-analyzer';

// These tests exercise the type checker against the *real* @hatchet-dev SDK
// (a devDependency), so they prove resolution through aliases / await /
// inference — not just string matching.

const here = path.dirname(fileURLToPath(import.meta.url));
const EXT_ROOT = path.resolve(here, '../../..');
// A virtual path under the extension root: picks up the extension tsconfig and
// resolves the SDK from node_modules. The file need not exist on disk — the
// analyzer overlays the provided text.
const SAMPLE = path.join(EXT_ROOT, '__vitest_type_sample.ts');

const analyzer = new WorkflowTypeAnalyzer();
const analyze = (src: string): WorkflowAnchor[] | null => analyzer.analyze(SAMPLE, src);
const byName = (anchors: WorkflowAnchor[] | null) =>
  new Map((anchors ?? []).map((a) => [a.name, a]));

const PRELUDE = `
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import type { WorkflowDeclaration } from '@hatchet-dev/typescript-sdk/v1';
const hatchet = HatchetClient.init();
type DagInput = { Message: string };
// Customer-style alias over the SDK type.
type DurableWorkflow = WorkflowDeclaration<DagInput, {}, {}>;
`;

describe('type-driven detection (real SDK)', () => {
  let anchors: ReturnType<typeof byName>;

  beforeAll(() => {
    anchors = byName(
      analyze(`${PRELUDE}
        // factory returning the aliased SDK type
        function createWorkflowBuilder(name: string): DurableWorkflow {
          return hatchet.workflow<DagInput>({ name });
        }
        function createWrapperWorkflow() {
          const wf = createWorkflowBuilder('wrapper');
          const fetchData = wf.durableTask({ name: 'fetch-data', fn: async () => ({}) });
          return wf;
        }
        const directWorkflow = hatchet.workflow<DagInput>({ name: 'direct' });
        const notWorkflow = { foo: 1 };
      `),
    );
  });

  it('anchors a function that returns a workflow type (through a customer alias)', () => {
    expect(anchors.get('createWorkflowBuilder')).toMatchObject({ kind: 'function' });
  });

  it('anchors a function whose inferred return type is a workflow', () => {
    expect(anchors.get('createWrapperWorkflow')).toMatchObject({ kind: 'function' });
  });

  it('dedupes a workflow variable declared inside a function anchor', () => {
    // `wf` (the await-resolved, alias-typed builder) lives inside
    // createWrapperWorkflow, which already represents that DAG.
    expect(anchors.has('wf')).toBe(false);
  });

  it('anchors a top-level inferred direct workflow (no annotation, no wrapper)', () => {
    expect(anchors.get('directWorkflow')).toMatchObject({ kind: 'variable' });
  });

  it('does not anchor the client, a plain object, or a task handle', () => {
    expect(anchors.has('hatchet')).toBe(false);
    expect(anchors.has('notWorkflow')).toBe(false);
    expect(anchors.has('fetchData')).toBe(false); // durableTask() returns a task opts, not a workflow
  });

  it('captures the enclosing function range for function anchors', () => {
    const fn = anchors.get('createWrapperWorkflow');
    expect(fn?.endLine).toBeGreaterThan(fn!.declarationLine);
  });
});

describe('non-workflow files', () => {
  it('returns an empty list (not null) when types resolve but nothing is a workflow', () => {
    const result = analyze(`export const x = 1; export function f() { return 'hi'; }`);
    expect(result).toEqual([]);
  });
});

describe('two-level wrapper: stub -> builder', () => {
  // A stub/descriptor flows into a builder that returns the SDK workflow; all
  // tasks live in the outer function. The stub layer must stay transparent.
  const anchors = byName(
    analyze(`${PRELUDE}
      interface WorkflowStub { name: string; version?: string }
      function createWorkflowStub(opts: WorkflowStub): WorkflowStub { return opts; }
      const agentStub = createWorkflowStub({ name: 'agent', version: 'v1' });

      async function createWorkflowBuilder(args: { stub: WorkflowStub; apiKey?: string }): Promise<DurableWorkflow> {
        return hatchet.workflow<DagInput>({ name: args.stub.name });
      }

      export async function createAgentWorkflow(args: { apiKey: string }): Promise<DurableWorkflow> {
        const builder = await createWorkflowBuilder({ stub: agentStub, apiKey: args.apiKey });
        const stream = builder.durableTask({ name: 'stream', fn: async () => ({}) });
        builder.durableTask({ name: 'persist', parents: [stream], fn: async () => ({}) });
        return builder;
      }
    `),
  );

  it('anchors the builder and the outer function through the stub layer', () => {
    expect(anchors.get('createWorkflowBuilder')).toMatchObject({ kind: 'function' });
    expect(anchors.get('createAgentWorkflow')).toMatchObject({ kind: 'function' });
  });

  it('does not anchor the stub descriptor (not a workflow type)', () => {
    expect(anchors.has('agentStub')).toBe(false);
  });

  it('dedupes the inner builder variable into the function', () => {
    expect(anchors.has('builder')).toBe(false);
  });
});

describe('customer fluent-builder shape: stub -> builder.task(...) -> builder.build()', () => {
  // A custom WorkflowBuilder with a positional .task() and a .build() that
  // returns the SDK workflow. The function returns the built workflow.
  const anchors = byName(
    analyze(`${PRELUDE}
      interface TaskRef {}
      interface WorkflowBuilder {
        task(name: string, opts: { parents?: TaskRef[]; fn: (a: any) => any }): TaskRef;
        build(): DurableWorkflow;
      }
      interface WorkflowStub { name: string }
      const stub: WorkflowStub = { name: 'my-workflow' };
      declare function createWorkflowBuilder(args: { stub: WorkflowStub }): Promise<WorkflowBuilder>;

      export async function createMyWorkflow(): Promise<DurableWorkflow> {
        const builder = await createWorkflowBuilder({ stub });
        const step1 = builder.task('step-1', { fn: async () => ({}) });
        builder.task('step-2', { parents: [step1], fn: async () => ({}) });
        return builder.build();
      }
    `),
  );

  it('anchors exactly the function (return type is the built workflow)', () => {
    expect(anchors.size).toBe(1);
    expect(anchors.get('createMyWorkflow')).toMatchObject({ kind: 'function' });
  });

  it('does not anchor the stub or the custom builder variable', () => {
    expect(anchors.has('stub')).toBe(false);
    expect(anchors.has('builder')).toBe(false);
  });
});

describe('composition wrapper: DurableWorkflow holds an SDK declaration', () => {
  // The wrapper does not *extend* an SDK class — it *holds* one as a field
  // (`_sdkDeclaration: BaseWorkflowDeclaration<…>`). Detection must recognise
  // that structurally.
  const anchors = byName(
    analyze(`
      import type { BaseWorkflowDeclaration, JsonObject } from '@hatchet-dev/typescript-sdk/v1';
      interface Out {}
      interface WorkflowTrigger<I extends JsonObject> { run: (i: I) => Promise<unknown> }
      interface WorkflowStub<I extends JsonObject> extends WorkflowTrigger<I> { readonly name: string }
      interface DurableWorkflow<I extends JsonObject> extends WorkflowTrigger<I> {
        readonly name: string;
        readonly _sdkDeclaration: BaseWorkflowDeclaration<I, Out>;
      }
      interface WorkflowBuilder<I extends JsonObject> {
        task: (name: string, opts: any) => unknown;
        build: () => DurableWorkflow<I>;
      }
      interface MyInput extends JsonObject { x: string }
      declare function createWorkflowStub<I extends JsonObject>(o: { name: string }): WorkflowStub<I>;
      declare function createWorkflowBuilder<I extends JsonObject>(a: { stub: WorkflowStub<I> }): Promise<WorkflowBuilder<I>>;
      export const myStub = createWorkflowStub<MyInput>({ name: 'w' });
      export async function createMyWorkflow(): Promise<DurableWorkflow<MyInput>> {
        const builder = await createWorkflowBuilder<MyInput>({ stub: myStub });
        const s1 = builder.task('s1', { fn: async () => ({}) });
        builder.task('s2', { parents: [s1], fn: async () => ({}) });
        return builder.build();
      }
    `),
  );

  it('anchors the function that returns the composing wrapper', () => {
    expect(anchors.get('createMyWorkflow')).toMatchObject({ kind: 'function' });
  });

  it('ignores the stub and the builder (neither composes an SDK declaration)', () => {
    expect(anchors.has('myStub')).toBe(false);
    expect(anchors.has('builder')).toBe(false);
  });

  it('produces exactly one anchor', () => {
    expect(anchors.size).toBe(1);
  });
});
