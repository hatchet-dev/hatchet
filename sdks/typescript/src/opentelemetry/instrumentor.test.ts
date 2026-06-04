/**
 * Tests for batch runWorkflows instrumentation:
 * - creates one hatchet.run_workflow span per item under the parent hatchet.run_workflows span
 * - injects each item's own traceparent into its additionalMetadata
 * - ends all started item spans (with error status) when span building,
 *   the underlying synchronous call, or the returned promise fails
 */

import {
  trace,
  propagation,
  context,
  ROOT_CONTEXT,
  SpanStatusCode,
  type Context,
  type ContextManager,
} from '@opentelemetry/api';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import {
  BasicTracerProvider,
  InMemorySpanExporter,
  SimpleSpanProcessor,
  type ReadableSpan,
} from '@opentelemetry/sdk-trace-base';

import { HatchetInstrumentor } from './instrumentor';

// Node has no context manager registered by default (unlike Python's
// contextvars), so `startActiveSpan` would not make the batch span the active
// context. Register a minimal synchronous stack-based manager so the legacy
// (flag-off) path can resolve the active span exactly as it does in production.
class StackContextManager implements ContextManager {
  private _stack: Context[] = [];

  active(): Context {
    return this._stack[this._stack.length - 1] ?? ROOT_CONTEXT;
  }

  with<A extends unknown[], F extends (...args: A) => ReturnType<F>>(
    ctx: Context,
    fn: F,
    thisArg?: ThisParameterType<F>,
    ...args: A
  ): ReturnType<F> {
    this._stack.push(ctx);
    try {
      return fn.call(thisArg as ThisParameterType<F>, ...args);
    } finally {
      this._stack.pop();
    }
  }

  bind<T>(_ctx: Context, target: T): T {
    return target;
  }

  enable(): this {
    return this;
  }

  disable(): this {
    this._stack = [];
    return this;
  }
}

const BATCH_SPAN_NAME = 'hatchet.run_workflows';
const ITEM_SPAN_NAME = 'hatchet.run_workflow';

type RunInput = {
  workflowName: string;
  input: unknown;
  options?: {
    additionalMetadata?: Record<string, string>;
    priority?: number;
    parentId?: string;
  };
};

function makeRun(name: string): RunInput {
  return {
    workflowName: name,
    input: { greeting: name },
    options: { additionalMetadata: { caller: 'test' } },
  };
}

function spansByName(exporter: InMemorySpanExporter, name: string): ReadableSpan[] {
  return exporter.getFinishedSpans().filter((s) => s.name === name);
}

describe('HatchetInstrumentor batch runWorkflows', () => {
  let exporter: InMemorySpanExporter;
  let provider: BasicTracerProvider;
  let instrumentor: HatchetInstrumentor;
  let prototype: { runWorkflows: (runs: RunInput[], batchSize?: number) => Promise<unknown> };
  let originalRunWorkflows: jest.Mock;

  beforeEach(() => {
    exporter = new InMemorySpanExporter();
    provider = new BasicTracerProvider({
      spanProcessors: [new SimpleSpanProcessor(exporter)],
    });
    trace.setGlobalTracerProvider(provider);
    propagation.setGlobalPropagator(new W3CTraceContextPropagator());
    context.setGlobalContextManager(new StackContextManager());

    instrumentor = new HatchetInstrumentor({
      enableHatchetCollector: false,
      individualRunSpansForBulkRun: true,
    });

    originalRunWorkflows = jest.fn(async (runs: RunInput[]) => runs.map(() => ({ id: 'ok' })));
    prototype = { runWorkflows: originalRunWorkflows };
    // _patchRunWorkflows is private; intentionally bypass typing for test access.
    (
      instrumentor as unknown as { _patchRunWorkflows: (p: typeof prototype) => void }
    )._patchRunWorkflows(prototype);
  });

  afterEach(async () => {
    await provider.shutdown();
    trace.disable();
    propagation.disable();
    context.disable();
  });

  it('creates one item span per config with a distinct, item-scoped traceparent', async () => {
    const runs = [makeRun('wf-0'), makeRun('wf-1'), makeRun('wf-2')];
    await prototype.runWorkflows(runs);

    const batchSpans = spansByName(exporter, BATCH_SPAN_NAME);
    const itemSpans = spansByName(exporter, ITEM_SPAN_NAME);
    expect(batchSpans).toHaveLength(1);
    expect(itemSpans).toHaveLength(3);
    for (const s of itemSpans) {
      expect(s.status.code).toBe(SpanStatusCode.UNSET);
    }

    // Each enhanced item must carry a traceparent matching ITS OWN item span
    // (not the batch span). This is the regression the PR fixes.
    const enhanced = originalRunWorkflows.mock.calls[0][0] as RunInput[];
    const traceparents = enhanced.map((r) => r.options?.additionalMetadata?.traceparent);
    expect(new Set(traceparents).size).toBe(3);

    const itemSpanIds = new Set(itemSpans.map((s) => s.spanContext().spanId));
    const batchSpanId = batchSpans[0].spanContext().spanId;
    for (const tp of traceparents) {
      expect(typeof tp).toBe('string');
      const [, , spanIdHex] = tp!.split('-');
      expect(itemSpanIds.has(spanIdHex)).toBe(true);
      expect(spanIdHex).not.toBe(batchSpanId);
    }
  });

  it('ends already-started item spans with error status when span construction fails partway', async () => {
    const { tracer } = instrumentor as unknown as { tracer: ReturnType<typeof trace.getTracer> };
    const realStartSpan = tracer.startSpan.bind(tracer);
    let call = 0;
    const startSpanSpy = jest
      .spyOn(tracer, 'startSpan')
      .mockImplementation((name: string, opts?: Parameters<typeof tracer.startSpan>[1]) => {
        if (name === ITEM_SPAN_NAME) {
          call += 1;
          if (call === 2) throw new Error('build boom');
        }
        return realStartSpan(name, opts);
      });

    const runs = [makeRun('wf-0'), makeRun('wf-1'), makeRun('wf-2')];

    await expect(prototype.runWorkflows(runs)).rejects.toThrow('build boom');
    expect(originalRunWorkflows).not.toHaveBeenCalled();

    const itemSpans = spansByName(exporter, ITEM_SPAN_NAME);
    // Only the first item span got created (second threw before startSpan returned).
    expect(itemSpans).toHaveLength(1);
    expect(itemSpans[0].status.code).toBe(SpanStatusCode.ERROR);

    const batchSpans = spansByName(exporter, BATCH_SPAN_NAME);
    expect(batchSpans).toHaveLength(1);
    expect(batchSpans[0].status.code).toBe(SpanStatusCode.ERROR);

    startSpanSpy.mockRestore();
  });

  it('ends item spans with error status when the wrapped call throws synchronously', async () => {
    originalRunWorkflows.mockImplementation(() => {
      throw new Error('sync boom');
    });

    const runs = [makeRun('wf-0'), makeRun('wf-1')];
    await expect(prototype.runWorkflows(runs)).rejects.toThrow('sync boom');

    const itemSpans = spansByName(exporter, ITEM_SPAN_NAME);
    expect(itemSpans).toHaveLength(2);
    for (const s of itemSpans) {
      expect(s.status.code).toBe(SpanStatusCode.ERROR);
    }
    const batchSpans = spansByName(exporter, BATCH_SPAN_NAME);
    expect(batchSpans).toHaveLength(1);
    expect(batchSpans[0].status.code).toBe(SpanStatusCode.ERROR);
  });

  it('ends item spans with error status when the returned promise rejects', async () => {
    originalRunWorkflows.mockRejectedValue(new Error('async boom'));

    const runs = [makeRun('wf-0'), makeRun('wf-1')];
    await expect(prototype.runWorkflows(runs)).rejects.toThrow('async boom');

    const itemSpans = spansByName(exporter, ITEM_SPAN_NAME);
    expect(itemSpans).toHaveLength(2);
    for (const s of itemSpans) {
      expect(s.status.code).toBe(SpanStatusCode.ERROR);
    }
    const batchSpans = spansByName(exporter, BATCH_SPAN_NAME);
    expect(batchSpans).toHaveLength(1);
    expect(batchSpans[0].status.code).toBe(SpanStatusCode.ERROR);
  });

  describe('with individualRunSpansForBulkRun disabled (default)', () => {
    let defaultInstrumentor: HatchetInstrumentor;
    let defaultPrototype: typeof prototype;
    let defaultOriginal: jest.Mock;

    beforeEach(() => {
      defaultInstrumentor = new HatchetInstrumentor({ enableHatchetCollector: false });
      defaultOriginal = jest.fn(async (runs: RunInput[]) => runs.map(() => ({ id: 'ok' })));
      defaultPrototype = { runWorkflows: defaultOriginal };
      (
        defaultInstrumentor as unknown as {
          _patchRunWorkflows: (p: typeof defaultPrototype) => void;
        }
      )._patchRunWorkflows(defaultPrototype);
    });

    it('creates no item spans and injects the batch span traceparent into every item', async () => {
      const runs = [makeRun('wf-0'), makeRun('wf-1'), makeRun('wf-2')];
      await defaultPrototype.runWorkflows(runs);

      const batchSpans = spansByName(exporter, BATCH_SPAN_NAME);
      expect(batchSpans).toHaveLength(1);
      // No per-item spans should be created when the flag is off — this preserves
      // the existing span structure for downstream collectors.
      expect(spansByName(exporter, ITEM_SPAN_NAME)).toHaveLength(0);

      const enhanced = defaultOriginal.mock.calls[0][0] as RunInput[];
      const traceparents = enhanced.map((r) => r.options?.additionalMetadata?.traceparent);
      // Every item carries the SAME traceparent, pointing at the batch span.
      expect(new Set(traceparents).size).toBe(1);
      const batchSpanId = batchSpans[0].spanContext().spanId;
      for (const tp of traceparents) {
        expect(typeof tp).toBe('string');
        const [, , spanIdHex] = tp!.split('-');
        expect(spanIdHex).toBe(batchSpanId);
      }
    });
  });
});
