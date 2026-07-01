import { describe, it, expect } from 'vitest';
import { detectTsWorkflowDeclarations, parseWorkflows } from '../workflow-parser';
import type { WorkflowFactoryAnnotation } from '../jsdoc-annotations';

const detect = (src: string, ann?: Map<string, WorkflowFactoryAnnotation>) =>
  detectTsWorkflowDeclarations(src, 'wf.ts', ann);
const tasksOf = (src: string, varName: string, ann?: Map<string, WorkflowFactoryAnnotation>) =>
  parseWorkflows(src, 'wf.ts', ann).find((w) => w.varName === varName)?.tasks ?? [];
const names = (tasks: { displayName: string }[]) => tasks.map((t) => t.displayName);

describe('syntactic detection — direct workflows', () => {
  it('detects a literal-named workflow with tasks and a parent edge', () => {
    const src = `
      const wf = hatchet.workflow({ name: 'lit' });
      const a = wf.task({ name: 'a' });
      const b = wf.task({ name: 'b', parents: [a] });
    `;
    const d = detect(src);
    expect(d).toHaveLength(1);
    expect(d[0]).toMatchObject({ varName: 'wf', name: 'lit' });

    const tasks = tasksOf(src, 'wf');
    expect(names(tasks)).toEqual(['a', 'b']);
    expect(tasks.find((t) => t.displayName === 'b')?.parentVarIds).toEqual(['a']);
  });

  it('keeps a non-literal name as the label (source text)', () => {
    const d = detect(`const wf = hatchet.workflow({ name: stub.name });`);
    expect(d).toHaveLength(1);
    expect(d[0].name).toBe('stub.name');
  });

  it('honours explicit generics on the workflow call', () => {
    const d = detect(`const wf = hatchet.workflow<TInput & JsonObject, Out>({ name: 'g' });`);
    expect(d.map((x) => x.varName)).toEqual(['wf']);
  });
});

describe('task-first detection — no annotation required', () => {
  it('treats any variable with durableTask calls as a workflow', () => {
    const src = `
      async function build() {
        const builder = await createWorkflowBuilder({ stub: s });
        const step1 = builder.durableTask('step-1', {});
        const step2 = builder.durableTask('step-2', { parents: [step1] });
        builder.durableTask('step-3', { parents: [step1, step2] });
      }
    `;
    const d = detect(src);
    expect(d.map((x) => x.varName)).toEqual(['builder']);

    const tasks = tasksOf(src, 'builder');
    expect(names(tasks)).toEqual(['step-1', 'step-2', 'step-3']);
    expect(tasks.find((t) => t.displayName === 'step-3')?.parentVarIds).toEqual(['step1', 'step2']);
  });

  it('parses both positional and options-object task forms', () => {
    const positional = tasksOf(
      `const wf = hatchet.workflow({ name: 'p' });
       const a = wf.task('a', {});
       const b = wf.task('b', { parents: [a] });`,
      'wf',
    );
    expect(names(positional)).toEqual(['a', 'b']);
    expect(positional[1].parentVarIds).toEqual(['a']);

    const options = tasksOf(
      `const wf = hatchet.workflow({ name: 'o' });
       const a = wf.task({ name: 'a' });
       const b = wf.task({ name: 'b', parents: [a] });`,
      'wf',
    );
    expect(names(options)).toEqual(['a', 'b']);
    expect(options[1].parentVarIds).toEqual(['a']);
  });

  it('excludes lifecycle handlers (onFailure / onSuccess) from the DAG', () => {
    const tasks = tasksOf(
      `const wf = hatchet.workflow({ name: 'h' });
       wf.task({ name: 'a' });
       wf.onFailure({ fn: f });
       wf.onSuccess({ fn: g });`,
      'wf',
    );
    expect(names(tasks)).toEqual(['a']);
  });

  it('does not treat unrelated methods as tasks', () => {
    expect(detect(`const x = foo.bar({ name: 'x' }); x.baz({ name: 'a' });`)).toHaveLength(0);
  });
});

describe('annotated factory usages', () => {
  const ann = new Map<string, WorkflowFactoryAnnotation>([
    ['createWorkflowBuilder', { functionName: 'createWorkflowBuilder', taskMethod: 'task', taskParentsProp: 'parents' }],
  ]);

  it('detects an awaited factory usage and reads the name from the call args', () => {
    const d = detect(`const wf = await createWorkflowBuilder({ stub: { name: 'wf-x' } });`, ann);
    expect(d).toHaveLength(1);
    expect(d[0]).toMatchObject({ varName: 'wf', name: 'wf-x' });
  });

  it('falls back to the variable name when the args carry no static name', () => {
    const d = detect(`const wf = await createWorkflowBuilder({ stub: someStub });`, ann);
    expect(d.map((x) => x.name)).toEqual(['wf']);
  });
});

describe('edge cases', () => {
  it('returns nothing for a file with no workflows', () => {
    expect(detect(`const x = 1; function f() { return 2; }`)).toEqual([]);
  });

  it('does not crash on malformed input', () => {
    expect(() => detect(`const wf = hatchet.workflow({ name: `)).not.toThrow();
  });
});
