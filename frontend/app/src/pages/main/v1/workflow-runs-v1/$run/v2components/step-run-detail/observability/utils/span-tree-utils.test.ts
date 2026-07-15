import {
  getSpanAttributeLabel,
  getSpanDisplayLabel,
  getSpanGroupLabel,
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

  test('engine step-run span shows the task name', () => {
    const span = node('hatchet.engine.start_step_run', {
      'hatchet.span_source': 'engine',
      'hatchet.task_name': 'process-message',
      'hatchet.step_name': 'process-message',
    });

    assert.strictEqual(getSpanDisplayLabel(span), 'process-message');
  });

  test('step-run span without a task name falls back to the step name', () => {
    const span = node('hatchet.start_step_run', {
      'hatchet.step_name': 'send-email',
    });

    assert.strictEqual(getSpanDisplayLabel(span), 'send-email');
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
