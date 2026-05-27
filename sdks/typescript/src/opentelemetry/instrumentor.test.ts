/**
 * Tests for batch runWorkflows instrumentation:
 * - creates one hatchet.run_workflow span per item under the parent hatchet.run_workflows span
 * - injects each item's own traceparent into its additionalMetadata
 * - ends all started item spans (with error status) when span building,
 *   the underlying synchronous call, or the returned promise fails
 */

import { trace, propagation, SpanStatusCode } from '@opentelemetry/api';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import {
  BasicTracerProvider,
  InMemorySpanExporter,
  SimpleSpanProcessor,
  type ReadableSpan,
} from '@opentelemetry/sdk-trace-base';

import { HatchetInstrumentor } from './instrumentor';

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

    instrumentor = new HatchetInstrumentor({ enableHatchetCollector: false });

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
});
