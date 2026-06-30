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
