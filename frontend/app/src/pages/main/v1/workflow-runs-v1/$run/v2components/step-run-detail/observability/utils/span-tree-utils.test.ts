import {
  getSpanAttributeLabel,
  getSpanDisplayLabel,
  getSpanGroupLabel,
  isSpanError,
} from './span-tree-utils';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import * as assert from 'node:assert';
import { describe, test } from 'node:test';

function node(
  spanName: string,
  spanAttributes: Record<string, string> = {},
): OtelSpanTree {
  return {
    spanId: 'span-1',
    parentSpanId: undefined,
    spanName,
    statusCode: 'OK' as OtelSpanTree['statusCode'],
    statusMessage: undefined,
    durationNs: 1_000_000,
    createdAt: '2025-01-01T00:00:00.000Z',
    spanAttributes,
    children: [],
  };
}

describe('getSpanDisplayLabel', () => {
  test('workflow-run spans that share a display name stay distinguishable by run id', () => {
    const shared = 'repro-child-1784075658981';
    const a = node('hatchet.engine.workflow_run', {
      'hatchet.workflow_name': shared,
      'hatchet.workflow_run_id': '11111111-aaaa-bbbb-cccc-222222222222',
    });
    const b = node('hatchet.engine.workflow_run', {
      'hatchet.workflow_name': shared,
      'hatchet.workflow_run_id': '33333333-dddd-eeee-ffff-444444444444',
    });

    assert.notStrictEqual(getSpanDisplayLabel(a), getSpanDisplayLabel(b));
    assert.strictEqual(getSpanDisplayLabel(a), `${shared} (11111111)`);
    assert.strictEqual(getSpanDisplayLabel(b), `${shared} (33333333)`);
  });

  test('a retried step-run row shows the task name with the retry number', () => {
    const base = node('hatchet.engine.start_step_run', {
      'hatchet.task_name': 'retry-repro',
      'hatchet.retry_count': '0',
    });
    const retry1 = node('hatchet.engine.start_step_run', {
      'hatchet.task_name': 'retry-repro',
      'hatchet.retry_count': '1',
    });
    const retry2 = node('hatchet.engine.start_step_run', {
      'hatchet.task_name': 'retry-repro',
      'hatchet.retry_count': '2',
    });

    assert.strictEqual(getSpanDisplayLabel(base), 'retry-repro');
    assert.strictEqual(getSpanDisplayLabel(retry1), 'retry-repro (retry 1)');
    assert.strictEqual(getSpanDisplayLabel(retry2), 'retry-repro (retry 2)');
  });

  test('step-run span without a task name falls back to the step name', () => {
    const span = node('hatchet.start_step_run', {
      'hatchet.step_name': 'send-email',
    });

    assert.strictEqual(getSpanDisplayLabel(span), 'send-email');
  });

  test('engine event rows show the event key with a short event id', () => {
    const received = node('hatchet.engine.event', {
      'hatchet.event_key': 'user:created',
      'hatchet.event_id': 'aaaaaaaa-1111-2222-3333-444444444444',
    });
    const emitted = node('hatchet.engine.event_emitted', {
      'hatchet.event_key': 'order:placed',
      'hatchet.event_id': '11111111-aaaa-bbbb-cccc-222222222222',
    });

    assert.strictEqual(
      getSpanDisplayLabel(received),
      'user:created (aaaaaaaa)',
    );
    assert.strictEqual(getSpanDisplayLabel(emitted), 'order:placed (11111111)');
  });

  test('application/custom spans keep their own span name', () => {
    const span = node('inventory.check-availability', {
      'my.custom.attr': 'value',
    });

    assert.strictEqual(
      getSpanDisplayLabel(span),
      'inventory.check-availability',
    );
  });
});

describe('getSpanGroupLabel', () => {
  test('workflow-run groups read as "workflow runs"', () => {
    assert.strictEqual(
      getSpanGroupLabel('hatchet.engine.workflow_run'),
      'workflow runs',
    );
  });

  test('step-run groups read as "tasks", engine and sdk alike', () => {
    assert.strictEqual(
      getSpanGroupLabel('hatchet.engine.start_step_run'),
      'tasks',
    );
    assert.strictEqual(getSpanGroupLabel('hatchet.start_step_run'), 'tasks');
  });

  test('emitted-event groups read as "emitted events"', () => {
    assert.strictEqual(
      getSpanGroupLabel('hatchet.engine.event_emitted'),
      'emitted events',
    );
  });

  test('unknown span names keep their own name', () => {
    assert.strictEqual(
      getSpanGroupLabel('inventory.check-availability'),
      'inventory.check-availability',
    );
  });
});

describe('getSpanAttributeLabel', () => {
  test('a custom span carries the propagated task/step name', () => {
    const span = node('inventory.check-availability', {
      'hatchet.step_name': 'process-message',
    });

    assert.strictEqual(getSpanAttributeLabel(span), 'process-message');
  });

  test('is undefined when no task or step name is present', () => {
    const span = node('hatchet.engine.workflow_run', {
      'hatchet.workflow_name': 'repro-parent',
    });

    assert.strictEqual(getSpanAttributeLabel(span), undefined);
  });
});

describe('isSpanError', () => {
  const errored = (spanName: string): OtelSpanTree => ({
    ...node(spanName),
    statusCode: 'ERROR' as OtelSpanTree['statusCode'],
  });

  test('an engine span with status OK is not an error even if a child failed', () => {
    const span: OtelSpanTree = {
      ...node('hatchet.engine.start_step_run', {
        'hatchet.span_source': 'engine',
      }),
      children: [errored('hatchet.engine.start_step_run')],
    };

    assert.strictEqual(isSpanError(span), false);
  });

  test('an engine span with status ERROR is an error', () => {
    const span: OtelSpanTree = {
      ...errored('hatchet.engine.start_step_run'),
      spanAttributes: { 'hatchet.span_source': 'engine' },
    };

    assert.strictEqual(isSpanError(span), true);
  });

  test('a non-engine span inherits an error from its subtree', () => {
    const span: OtelSpanTree = {
      ...node('inventory.check-availability'),
      children: [errored('db.query')],
    };

    assert.strictEqual(isSpanError(span), true);
  });
});
