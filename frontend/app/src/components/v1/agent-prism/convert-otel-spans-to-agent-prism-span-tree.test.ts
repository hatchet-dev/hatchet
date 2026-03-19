import { convertOtelSpansToOtelSpanTree } from './convert-otel-spans-to-agent-prism-span-tree';
import type { RelevantOpenTelemetrySpanProperties } from './span-tree-type';
import * as assert from 'node:assert';
import { describe, test } from 'node:test';

const OtelStatusCode = { UNSET: 'UNSET', OK: 'OK', ERROR: 'ERROR' } as const;

type Span = RelevantOpenTelemetrySpanProperties;

function span(overrides: Partial<Span> & { spanId: string }): Span {
  return {
    spanName: 'test',
    parentSpanId: undefined,
    statusCode: OtelStatusCode.OK as Span['statusCode'],
    durationNs: 1_000_000,
    createdAt: '2025-01-01T00:00:00.000Z',
    spanAttributes: {},
    ...overrides,
  };
}

function engineQueued(
  stepRunId: string,
  parentSpanId: string,
  extra?: Partial<Span>,
): Span {
  return span({
    spanId: `eq_${stepRunId}`,
    parentSpanId,
    spanName: 'hatchet.engine.queued',
    spanAttributes: {
      'hatchet.span_source': 'engine',
      'hatchet.step_run_id': stepRunId,
      'hatchet.step_name': stepRunId,
      ...extra?.spanAttributes,
    },
    ...extra,
  });
}

function engineStepRun(
  stepRunId: string,
  parentSpanId: string,
  extra?: Partial<Span>,
): Span {
  return span({
    spanId: `esr_${stepRunId}`,
    parentSpanId,
    spanName: 'hatchet.start_step_run',
    spanAttributes: {
      'hatchet.span_source': 'engine',
      'hatchet.step_run_id': stepRunId,
      'hatchet.step_name': stepRunId,
      ...extra?.spanAttributes,
    },
    ...extra,
  });
}

function sdkStepRun(
  stepRunId: string,
  parentSpanId: string,
  extra?: Partial<Span>,
): Span {
  return span({
    spanId: `ssr_${stepRunId}`,
    parentSpanId,
    spanName: 'hatchet.start_step_run',
    spanAttributes: {
      'hatchet.step_run_id': stepRunId,
      'hatchet.step_name': stepRunId,
      ...extra?.spanAttributes,
    },
    ...extra,
  });
}

function sdkRunWorkflow(
  spanId: string,
  parentSpanId: string,
  taskName: string,
  stepRunId: string,
): Span {
  return span({
    spanId,
    parentSpanId,
    spanName: 'hatchet.run_workflow',
    spanAttributes: {
      'hatchet.task_name': taskName,
      'hatchet.step_name': 'dag-confirmation',
      'hatchet.step_run_id': stepRunId,
    },
  });
}

function sdkUserSpan(
  spanId: string,
  parentSpanId: string,
  name: string,
  stepRunId: string,
): Span {
  return span({
    spanId,
    parentSpanId,
    spanName: name,
    statusCode: OtelStatusCode.UNSET as Span['statusCode'],
    spanAttributes: {
      'hatchet.step_run_id': stepRunId,
      'hatchet.step_name': stepRunId,
    },
  });
}

function collectChildNames(tree: ReturnType<typeof convertOtelSpansToOtelSpanTree>) {
  const root = tree[0];
  return root.children.map((c) => ({
    name: c.spanAttributes?.['hatchet.step_name'] ?? c.spanName,
    spanId: c.spanId,
    childCount: c.children.length,
  }));
}

function asNonEmpty(spans: Span[]): [Span, ...Span[]] {
  assert.ok(spans.length > 0, 'spans must not be empty');
  return spans as [Span, ...Span[]];
}

const ROOT = 'root_span';
const ROOT_PARENT = undefined;

function rootSpan(): Span {
  return span({
    spanId: ROOT,
    parentSpanId: ROOT_PARENT,
    spanName: 'hatchet.start_workflow',
    spanAttributes: {
      instrumentor: 'hatchet',
      'hatchet.action_id': 'otel-order-processing:validate-order',
    },
    durationNs: 10_000_000,
  });
}

describe('convertOtelSpansToOtelSpanTree', () => {
  describe('deduplication', () => {
    test('removes engine start_step_run when SDK version exists', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('sr1', ROOT),
        engineStepRun('sr1', ROOT),
        sdkStepRun('sr1', ROOT),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];
      const stepRuns = root.children.filter(
        (c) => c.spanName === 'hatchet.start_step_run',
      );

      assert.strictEqual(stepRuns.length, 1, 'should have exactly one step run');
      assert.ok(
        !stepRuns[0].spanAttributes?.['hatchet.span_source'],
        'surviving span should be SDK (no span_source)',
      );
    });

    test('keeps engine start_step_run when no SDK version exists', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('sr1', ROOT),
        engineStepRun('sr1', ROOT),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];
      const stepRuns = root.children.filter(
        (c) => c.spanName === 'hatchet.start_step_run',
      );

      assert.strictEqual(stepRuns.length, 1);
      assert.strictEqual(
        stepRuns[0].spanAttributes?.['hatchet.span_source'],
        'engine',
      );
    });
  });

  describe('queued span merging', () => {
    test('merges queued span into step_run as queuedPhase', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('sr1', ROOT),
        engineStepRun('sr1', ROOT),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];
      const stepRun = root.children.find(
        (c) => c.spanName === 'hatchet.start_step_run',
      );

      assert.ok(stepRun, 'step run should exist');
      assert.ok(stepRun.queuedPhase, 'should have queuedPhase');
      assert.strictEqual(stepRun.queuedPhase.spanName, 'hatchet.engine.queued');
    });
  });

  describe('orphan engine queued spans with missing parents', () => {
    test('drops orphan queued spans with unique missing parents', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('sr1', ROOT),
        engineStepRun('sr1', ROOT),
        engineQueued('notif1', 'nonexistent_run_workflow_span'),
        engineQueued('notif2', 'another_nonexistent_span'),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      const allNames = root.children.map(
        (c) => c.spanAttributes?.['hatchet.step_run_id'],
      );
      assert.ok(!allNames.includes('notif1'), 'notif1 should be dropped');
      assert.ok(!allNames.includes('notif2'), 'notif2 should be dropped');
    });

    test('keeps queued spans whose missing parent is shared by siblings (implicit trace root)', () => {
      const IMPLICIT_ROOT = 'implicit_trace_root';
      const spans = asNonEmpty([
        engineQueued('validate', IMPLICIT_ROOT),
        engineStepRun('validate', IMPLICIT_ROOT),
        engineQueued('charge', IMPLICIT_ROOT),
        engineStepRun('charge', IMPLICIT_ROOT),
        engineQueued('reserve', IMPLICIT_ROOT),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      const reserveSpan = root.children.find(
        (c) => c.spanAttributes?.['hatchet.step_run_id'] === 'reserve',
      );
      assert.ok(reserveSpan, 'reserve should be synthesized, not dropped');
      assert.strictEqual(reserveSpan.spanName, 'hatchet.start_step_run');
      assert.strictEqual(reserveSpan.inProgress, true);
    });

    test('synthesizes in-progress span for queued span whose parent exists', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('sr1', ROOT),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];
      const synthetic = root.children.find(
        (c) =>
          c.spanAttributes?.['hatchet.step_run_id'] === 'sr1' &&
          c.spanName === 'hatchet.start_step_run',
      );

      assert.ok(synthetic, 'should synthesize in-progress span');
      assert.strictEqual(synthetic.inProgress, true);
      assert.ok(synthetic.queuedPhase, 'should have queuedPhase');
    });

    test('orphan SDK span adopted via reparentOrphans when synthetic step_run exists', () => {
      const IMPLICIT_ROOT = 'implicit_trace_root';
      const spans = asNonEmpty([
        engineQueued('validate', IMPLICIT_ROOT),
        engineStepRun('validate', IMPLICIT_ROOT),
        engineQueued('reserve', IMPLICIT_ROOT),
        sdkUserSpan('inv-check', 'missing_sdk_reserve_span', 'inventory.check-availability', 'reserve'),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      const reserveSpan = root.children.find(
        (c) => c.spanAttributes?.['hatchet.step_run_id'] === 'reserve',
      );
      assert.ok(reserveSpan, 'reserve synthetic span should exist');
      assert.ok(
        reserveSpan.children.some((c) => c.spanName === 'inventory.check-availability'),
        'inventory.check-availability should be reparented under reserve',
      );

      const rootIds = root.children.map((c) => c.spanId);
      assert.ok(!rootIds.includes('inv-check'), 'inventory span should not be at root');
    });
  });

  describe('reparentOrphans', () => {
    test('reparents orphan run_workflow under engine step_run with same step_run_id', () => {
      const dagStepRunId = 'dag-conf';
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued(dagStepRunId, ROOT),
        engineStepRun(dagStepRunId, ROOT),
        sdkRunWorkflow('rw1', 'nonexistent_sdk_dag_span', 'otel-send-notification', dagStepRunId),
        sdkRunWorkflow('rw2', 'nonexistent_sdk_dag_span', 'otel-other-task', dagStepRunId),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      assert.strictEqual(root.children.length, 1, 'root should have 1 child (dag step run)');

      const dagSpan = root.children[0];
      assert.strictEqual(
        dagSpan.spanAttributes?.['hatchet.step_run_id'],
        dagStepRunId,
      );
      assert.strictEqual(dagSpan.children.length, 2, 'dag span should have 2 reparented children');

      const childNames = dagSpan.children.map(
        (c) => c.spanAttributes?.['hatchet.task_name'],
      );
      assert.ok(childNames.includes('otel-send-notification'));
      assert.ok(childNames.includes('otel-other-task'));
    });

    test('orphan SDK user spans reparented under step_run with same step_run_id', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('sr1', ROOT),
        engineStepRun('sr1', ROOT),
        sdkUserSpan('user1', 'nonexistent_sdk_step_run', 'notification.render-template', 'sr1'),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      const stepRun = root.children.find(
        (c) => c.spanAttributes?.['hatchet.step_run_id'] === 'sr1',
      );
      assert.ok(stepRun);
      const renderTemplate = stepRun.children.find(
        (c) => c.spanName === 'notification.render-template',
      );
      assert.ok(renderTemplate, 'render-template should be reparented under step run');
    });
  });

  describe('poll progression stability', () => {
    const parentSpan = ROOT;

    test('poll 1: engine-only spans produce stable tree', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('validate', parentSpan),
        engineStepRun('validate', parentSpan),
        engineQueued('reserve', parentSpan),
        engineQueued('charge', parentSpan),
        engineStepRun('charge', parentSpan),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      assert.ok(root.children.length <= 3, 'should have at most 3 children');
      const names = root.children.map(
        (c) => c.spanAttributes?.['hatchet.step_name'],
      );
      assert.ok(names.includes('validate'));
      assert.ok(names.includes('charge'));
    });

    test('real task spans retain queuedPhase but are not synthetic roots', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('validate', ROOT),
        engineStepRun('validate', ROOT),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];
      const validateSpan = root.children.find(
        (c) => c.spanAttributes?.['hatchet.step_run_id'] === 'validate',
      );

      assert.ok(validateSpan, 'validate span should exist');
      assert.ok(validateSpan.queuedPhase, 'should have queuedPhase');
      assert.ok(
        !validateSpan.spanId.startsWith('__synthetic_'),
        'real task span should not have synthetic spanId',
      );
    });

    test('poll transition: trace mode should not surface standalone queued rows', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineStepRun('validate', parentSpan),
        engineQueued('reserve', parentSpan),
        engineQueued('charge', parentSpan),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans, undefined, undefined, {
        enableTraceInProgressSynthesis: false,
      });
      const root = tree[0];

      const queuedRows = root.children.filter(
        (c) => c.spanName === 'hatchet.engine.queued',
      );
      assert.strictEqual(
        queuedRows.length,
        0,
        'trace mode should not render standalone queued rows',
      );
    });

    test('poll 2: engine + SDK spans, orphan child-workflow queued spans dropped but dag-conf kept', () => {
      const spans = asNonEmpty([
        rootSpan(),
        engineQueued('validate', parentSpan),
        engineStepRun('validate', parentSpan),
        sdkStepRun('validate', parentSpan),
        engineQueued('reserve', parentSpan),
        engineQueued('charge', parentSpan),
        engineStepRun('charge', parentSpan),
        sdkStepRun('charge', parentSpan),
        engineQueued('dag-conf', parentSpan),
        engineQueued('notif1', 'missing_rw_span_1'),
        engineQueued('notif2', 'missing_rw_span_2'),
        engineQueued('other-task', 'missing_rw_span_3'),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      const childStepRunIds = root.children.map(
        (c) => c.spanAttributes?.['hatchet.step_run_id'],
      );

      assert.ok(!childStepRunIds.includes('notif1'), 'notif1 should be dropped');
      assert.ok(!childStepRunIds.includes('notif2'), 'notif2 should be dropped');
      assert.ok(!childStepRunIds.includes('other-task'), 'other-task should be dropped');

      assert.ok(childStepRunIds.includes('validate'));
      assert.ok(childStepRunIds.includes('charge'));
      assert.ok(childStepRunIds.includes('reserve'), 'reserve should be synthesized');
      assert.ok(childStepRunIds.includes('dag-conf'), 'dag-conf should be synthesized');
    });

    test('poll 3: child-workflow engine spans suppressed when parent run_workflow missing', () => {
      const TRACE_ROOT = 'implicit_trace_root';
      const RW_OTHER = 'rw_otel_other_task';

      const spans = asNonEmpty([
        engineQueued('validate', TRACE_ROOT),
        engineStepRun('validate', TRACE_ROOT),
        sdkStepRun('validate', TRACE_ROOT),
        sdkUserSpan('schema', 'ssr_validate', 'order.validate.schema', 'validate'),
        sdkUserSpan('fraud', 'ssr_validate', 'order.validate.fraud-check', 'validate'),

        engineQueued('reserve', TRACE_ROOT),
        engineStepRun('reserve', TRACE_ROOT),
        sdkUserSpan('inv-check', 'missing_sdk_reserve', 'inventory.check-availability', 'reserve'),

        engineQueued('charge', TRACE_ROOT),
        engineStepRun('charge', TRACE_ROOT),
        sdkStepRun('charge', TRACE_ROOT),
        sdkUserSpan('payment', 'ssr_charge', 'payment.process', 'charge'),

        engineQueued('dag-conf', TRACE_ROOT),

        engineQueued('notif1', 'rw_notif1'),
        engineQueued('notif2', 'rw_notif2'),

        engineQueued('other-task', RW_OTHER),
        engineStepRun('other-task', RW_OTHER),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      const childStepRunIds = root.children.map(
        (c) => c.spanAttributes?.['hatchet.step_run_id'],
      );

      assert.strictEqual(root.children.length, 4, 'root should have 4 children, not 5');
      assert.ok(childStepRunIds.includes('validate'));
      assert.ok(childStepRunIds.includes('reserve'));
      assert.ok(childStepRunIds.includes('charge'));
      assert.ok(childStepRunIds.includes('dag-conf'), 'dag-conf should be synthesized');
      assert.ok(!childStepRunIds.includes('other-task'), 'other-task should not be at root');
      assert.ok(!childStepRunIds.includes('notif1'), 'notif1 should not be at root');
      assert.ok(!childStepRunIds.includes('notif2'), 'notif2 should not be at root');
    });

    test('poll 4: SDK run_workflow spans arrive, reparented under dag engine step_run', () => {
      const dagStepRunId = 'dag-conf';
      const sdkDagSpanId = 'sdk_dag_span_id';

      const spans = asNonEmpty([
        rootSpan(),
        sdkStepRun('validate', parentSpan),
        sdkStepRun('charge', parentSpan),
        sdkStepRun('reserve', parentSpan),
        engineQueued(dagStepRunId, parentSpan),
        engineStepRun(dagStepRunId, parentSpan),
        sdkRunWorkflow('rw1', sdkDagSpanId, 'otel-send-notification', dagStepRunId),
        sdkRunWorkflow('rw2', sdkDagSpanId, 'otel-send-notification', dagStepRunId),
        sdkRunWorkflow('rw3', sdkDagSpanId, 'otel-other-task', dagStepRunId),
        engineQueued('notif1', 'rw1'),
        engineQueued('notif2', 'rw2'),
        engineQueued('other', 'rw3'),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      assert.strictEqual(root.children.length, 4, 'root should have 4 children');

      const dagSpan = root.children.find(
        (c) => c.spanAttributes?.['hatchet.step_run_id'] === dagStepRunId,
      );
      assert.ok(dagSpan, 'dag-conf should be present');
      assert.ok(dagSpan.children.length >= 3, 'dag should have run_workflow children');

      const rootStepRunIds = root.children.map(
        (c) => c.spanAttributes?.['hatchet.step_run_id'],
      );
      assert.ok(!rootStepRunIds.includes('notif1'), 'notif1 should not be at root');
      assert.ok(!rootStepRunIds.includes('notif2'), 'notif2 should not be at root');
    });

    test('orphan run_workflow spans suppressed when step_run not yet present', () => {
      const dagStepRunId = 'dag-conf';
      const sdkDagSpanId = 'sdk_dag_span_id';

      const spans = asNonEmpty([
        rootSpan(),
        sdkStepRun('validate', parentSpan),
        sdkStepRun('charge', parentSpan),
        sdkStepRun('reserve', parentSpan),
        sdkRunWorkflow('rw1', sdkDagSpanId, 'otel-send-notification', dagStepRunId),
        sdkRunWorkflow('rw2', sdkDagSpanId, 'otel-send-notification', dagStepRunId),
        sdkRunWorkflow('rw3', sdkDagSpanId, 'otel-other-task', dagStepRunId),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      assert.strictEqual(root.children.length, 3, 'root should have 3 children (no orphan run_workflows)');

      const childNames = root.children.map(
        (c) => c.spanAttributes?.['hatchet.step_run_id'],
      );
      assert.ok(childNames.includes('validate'));
      assert.ok(childNames.includes('charge'));
      assert.ok(childNames.includes('reserve'));
    });

    test('final poll: SDK dag-confirmation arrives, tree fully nested and stable', () => {
      const dagStepRunId = 'dag-conf';
      const sdkDagSpanId = 'sdk_dag_span_id';

      const spans = asNonEmpty([
        rootSpan(),
        sdkStepRun('validate', parentSpan),
        sdkStepRun('charge', parentSpan),
        sdkStepRun('reserve', parentSpan),
        engineQueued(dagStepRunId, parentSpan),
        sdkStepRun(dagStepRunId, parentSpan, { spanId: sdkDagSpanId }),
        sdkRunWorkflow('rw1', sdkDagSpanId, 'otel-send-notification', dagStepRunId),
        sdkRunWorkflow('rw2', sdkDagSpanId, 'otel-send-notification', dagStepRunId),
        sdkRunWorkflow('rw3', sdkDagSpanId, 'otel-other-task', dagStepRunId),
      ]);

      const tree = convertOtelSpansToOtelSpanTree(spans);
      const root = tree[0];

      assert.strictEqual(root.children.length, 4, 'root should have 4 children');

      const dagSpan = root.children.find(
        (c) => c.spanId === sdkDagSpanId,
      );
      assert.ok(dagSpan, 'SDK dag span should be the surviving one');
      assert.strictEqual(dagSpan.children.length, 3, 'dag should have 3 run_workflow children');
    });
  });
});
