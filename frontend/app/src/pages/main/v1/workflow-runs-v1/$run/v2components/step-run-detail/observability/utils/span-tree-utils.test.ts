import {
  getSpanAttributeLabel,
  getSpanIdentityLabel,
  getSpanIdentityParts,
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

describe('getSpanIdentityParts', () => {
  test('a workflow-run span gets the workflow name with a short run id discriminator', () => {
    const span = node('hatchet.engine.workflow_run', {
      'hatchet.workflow_name': 'repro-child-1784075658981',
      'hatchet.workflow_run_id': '11111111-aaaa-bbbb-cccc-222222222222',
    });

    assert.deepStrictEqual(getSpanIdentityParts(span), {
      label: 'repro-child-1784075658981',
      discriminator: '(11111111)',
    });
  });

  test('a retried step-run span gets a retry discriminator, the base attempt none', () => {
    const base = node('hatchet.engine.start_step_run', {
      'hatchet.task_name': 'retry-repro',
      'hatchet.retry_count': '0',
    });
    const retry2 = node('hatchet.engine.start_step_run', {
      'hatchet.task_name': 'retry-repro',
      'hatchet.retry_count': '2',
    });

    assert.deepStrictEqual(getSpanIdentityParts(base), {
      label: 'retry-repro',
      discriminator: undefined,
    });
    assert.deepStrictEqual(getSpanIdentityParts(retry2), {
      label: 'retry-repro',
      discriminator: '(retry 2)',
    });
  });

  test('a step-run span without a task name falls back to the step name', () => {
    const span = node('hatchet.start_step_run', {
      'hatchet.step_name': 'send-email',
    });

    assert.strictEqual(getSpanIdentityParts(span)?.label, 'send-email');
  });

  test('engine event spans get the event key with a short event id discriminator', () => {
    const received = node('hatchet.engine.event', {
      'hatchet.event_key': 'user:created',
      'hatchet.event_id': 'aaaaaaaa-1111-2222-3333-444444444444',
    });
    const emitted = node('hatchet.engine.event_emitted', {
      'hatchet.event_key': 'order:placed',
      'hatchet.event_id': '11111111-aaaa-bbbb-cccc-222222222222',
    });

    assert.deepStrictEqual(getSpanIdentityParts(received), {
      label: 'user:created',
      discriminator: '(aaaaaaaa)',
    });
    assert.deepStrictEqual(getSpanIdentityParts(emitted), {
      label: 'order:placed',
      discriminator: '(11111111)',
    });
  });

  test('application/custom spans have no identity, even with task context attributes', () => {
    const span = node('inventory.check-availability', {
      'hatchet.task_name': 'process-message',
      'hatchet.step_name': 'process-message',
    });

    assert.strictEqual(getSpanIdentityParts(span), undefined);
  });
});

describe('getSpanIdentityLabel', () => {
  test('joins the label and discriminator for tooltip text', () => {
    const withDiscriminator = node('hatchet.engine.workflow_run', {
      'hatchet.workflow_name': 'repro-parent',
      'hatchet.workflow_run_id': '11111111-aaaa-bbbb-cccc-222222222222',
    });
    const withoutDiscriminator = node('hatchet.engine.start_step_run', {
      'hatchet.task_name': 'retry-repro',
      'hatchet.retry_count': '0',
    });

    assert.strictEqual(
      getSpanIdentityLabel(withDiscriminator),
      'repro-parent (11111111)',
    );
    assert.strictEqual(
      getSpanIdentityLabel(withoutDiscriminator),
      'retry-repro',
    );
    assert.strictEqual(
      getSpanIdentityLabel(node('inventory.check-availability')),
      undefined,
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
