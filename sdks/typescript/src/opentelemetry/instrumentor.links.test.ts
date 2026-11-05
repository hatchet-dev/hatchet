/**
 * Tests for HatchetInstrumentor.useLinksInsteadOfParent option.
 *
 * Verifies that fire-and-forget child runs use OTel span links instead of
 * parent-child relationships when the predicate returns true, while
 * preserving the default parent-child behaviour when it returns false.
 *
 * These tests use jest module mocks so they live in a separate file from
 * instrumentor.test.ts, which uses a real OTel SDK provider.
 */

// Minimal OTel span mock
const makeMockSpan = () => ({
  end: jest.fn(),
  recordException: jest.fn(),
  setStatus: jest.fn(),
  spanContext: jest.fn(() => ({
    traceId: 'a'.repeat(32),
    spanId: 'b'.repeat(16),
    traceFlags: 1,
  })),
});

// Build a mock tracer whose startActiveSpan captures its arguments.
const makeTracerMock = (span = makeMockSpan()) => {
  const calls: IArguments[] = [];
  const tracer = {
    _calls: calls,
    _span: span,
    startActiveSpan: jest.fn(function (...args: unknown[]) {
      // The last argument is always the callback (fn).
      const fn = args[args.length - 1] as (span: unknown) => unknown;
      return fn(span);
    }),
  };
  return tracer;
};

// Dummy action used in all tests.
const makeAction = (actionId = 'my-worker:my-task') => ({
  actionId,
  tenantId: 'tenant-1',
  workflowRunId: 'run-1',
  taskId: 'task-1',
  taskRunExternalId: 'ext-1',
  retryCount: 0,
  parentWorkflowRunId: undefined,
  childWorkflowIndex: undefined,
  childWorkflowKey: undefined,
  actionPayload: '{}',
  jobName: 'my-task',
  taskName: 'my-task',
  workflowId: 'wf-1',
  workflowVersionId: 'wfv-1',
  // Encode a fake traceparent in the metadata so extractContext finds a valid context.
  additionalMetadata: JSON.stringify({
    traceparent: '00-' + 'a'.repeat(32) + '-' + 'b'.repeat(16) + '-01',
  }),
});

// ---------------------------------------------------------------------------
// Sentinel object used as ROOT_CONTEXT in the mock — allows tests to assert
// that the implementation explicitly passes ROOT_CONTEXT, not the active ctx.
// ---------------------------------------------------------------------------
const MOCK_ROOT_CONTEXT = { _isRootContext: true } as const;

jest.mock('@opentelemetry/api', () => {
  const validSpanCtx = {
    traceId: 'a'.repeat(32),
    spanId: 'b'.repeat(16),
    traceFlags: 1,
  };

  // SpanKind.CONSUMER = 4, SpanStatusCode.OK = 1, SpanStatusCode.ERROR = 2
  return {
    SpanKind: { INTERNAL: 0, SERVER: 1, CLIENT: 2, PRODUCER: 3, CONSUMER: 4 },
    SpanStatusCode: { UNSET: 0, OK: 1, ERROR: 2 },
    ROOT_CONTEXT: MOCK_ROOT_CONTEXT,
    context: {
      active: jest.fn(() => ({})),
      with: jest.fn((_ctx: unknown, fn: () => unknown) => fn()),
    },
    propagation: {
      extract: jest.fn(() => ({ _extracted: true })),
      inject: jest.fn(),
    },
    trace: {
      getSpanContext: jest.fn(() => validSpanCtx),
      isSpanContextValid: jest.fn(() => true),
    },
    diag: {
      debug: jest.fn(),
      info: jest.fn(),
      warn: jest.fn(),
      error: jest.fn(),
    },
  };
});

jest.mock('@opentelemetry/instrumentation', () => {
  class InstrumentationBase {
    protected tracer: ReturnType<typeof makeTracerMock>;
    protected config: Record<string, unknown>;
    constructor(_name: string, _version: string, config: Record<string, unknown>) {
      this.config = config;
      this.tracer = makeTracerMock();
    }
    getConfig() {
      return this.config;
    }
    setConfig(cfg: Record<string, unknown>) {
      this.config = cfg;
    }
    protected _wrap(
      proto: Record<string, unknown>,
      method: string,
      wrapper: (orig: unknown) => unknown
    ) {
      proto[method] = wrapper(proto[method]);
    }
    protected _unwrap(_proto: Record<string, unknown>, _method: string) {
      // no-op in tests
    }
  }

  return {
    InstrumentationBase,
    InstrumentationNodeModuleDefinition: jest.fn(() => ({})),
    InstrumentationNodeModuleFile: jest.fn(() => ({})),
    isWrapped: jest.fn(() => false),
  };
});

// Import AFTER mocks are set up.
import { HatchetInstrumentor } from './instrumentor';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function buildWorkerProto() {
  return {
    workerId: 'worker-1',
    handleStartStepRun: jest.fn().mockResolvedValue(undefined),
  };
}

/**
 * Returns the startActiveSpan call arguments for the hatchet.start_step_run span.
 * startActiveSpan is always called with 4 arguments: (name, opts, context, fn).
 * In parent-child mode, context is the extracted parentContext.
 * In link mode, context is ROOT_CONTEXT and opts includes a non-empty links array.
 */
function getStartStepRunArgs(tracer: ReturnType<typeof makeTracerMock>) {
  const call = (tracer.startActiveSpan as jest.Mock).mock.calls.find(
    ([name]: [string]) => typeof name === 'string' && name.startsWith('hatchet.start_step_run')
  );
  expect(call).toBeDefined();
  return call as unknown[];
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('HatchetInstrumentor.useLinksInsteadOfParent', () => {
  it('uses parent-child semantics by default (no option provided)', async () => {
    const instrumentor = new HatchetInstrumentor({});
    const tracer = (instrumentor as unknown as { tracer: ReturnType<typeof makeTracerMock> }).tracer;

    const proto = buildWorkerProto();
    (instrumentor as unknown as { patchWorker: (e: unknown) => void }).patchWorker({
      InternalWorker: { prototype: proto },
    });

    await proto.handleStartStepRun(makeAction());

    const args = getStartStepRunArgs(tracer);
    // 4 args: (name, opts, parentContext, fn)
    expect(args).toHaveLength(4);
    // opts should NOT include links
    const opts = args[1] as Record<string, unknown>;
    expect(opts.links).toBeUndefined();
    // context is the extracted parent context, not ROOT_CONTEXT
    expect(args[2]).not.toBe(MOCK_ROOT_CONTEXT);
  });

  it('uses parent-child semantics when predicate returns false', async () => {
    const instrumentor = new HatchetInstrumentor({
      useLinksInsteadOfParent: () => false,
    });
    const tracer = (instrumentor as unknown as { tracer: ReturnType<typeof makeTracerMock> }).tracer;

    const proto = buildWorkerProto();
    (instrumentor as unknown as { patchWorker: (e: unknown) => void }).patchWorker({
      InternalWorker: { prototype: proto },
    });

    await proto.handleStartStepRun(makeAction());

    const args = getStartStepRunArgs(tracer);
    expect(args).toHaveLength(4);
    const opts = args[1] as Record<string, unknown>;
    expect(opts.links).toBeUndefined();
    expect(args[2]).not.toBe(MOCK_ROOT_CONTEXT);
  });

  it('uses span links with ROOT_CONTEXT when predicate returns true', async () => {
    const instrumentor = new HatchetInstrumentor({
      useLinksInsteadOfParent: () => true,
    });
    const tracer = (instrumentor as unknown as { tracer: ReturnType<typeof makeTracerMock> }).tracer;

    const proto = buildWorkerProto();
    (instrumentor as unknown as { patchWorker: (e: unknown) => void }).patchWorker({
      InternalWorker: { prototype: proto },
    });

    await proto.handleStartStepRun(makeAction());

    const args = getStartStepRunArgs(tracer);
    // 4 args: (name, opts, ROOT_CONTEXT, fn)
    expect(args).toHaveLength(4);
    // opts must include a non-empty links array
    const opts = args[1] as Record<string, unknown>;
    expect(Array.isArray(opts.links)).toBe(true);
    expect((opts.links as unknown[]).length).toBe(1);
    // context must be ROOT_CONTEXT, not the extracted parent context
    expect(args[2]).toBe(MOCK_ROOT_CONTEXT);
  });

  it('passes the actionId to the predicate', async () => {
    const predicate = jest.fn(() => false);
    const action = makeAction('custom-worker:custom-task');

    const instrumentor = new HatchetInstrumentor({
      useLinksInsteadOfParent: predicate,
    });

    const proto = buildWorkerProto();
    (instrumentor as unknown as { patchWorker: (e: unknown) => void }).patchWorker({
      InternalWorker: { prototype: proto },
    });

    await proto.handleStartStepRun(action);

    expect(predicate).toHaveBeenCalledWith('custom-worker:custom-task');
  });

  it('uses ROOT_CONTEXT with an empty links array when parent span context is invalid', async () => {
    const otelApi = require('@opentelemetry/api');
    jest.spyOn(otelApi.trace, 'isSpanContextValid').mockReturnValueOnce(false);

    const instrumentor = new HatchetInstrumentor({
      useLinksInsteadOfParent: () => true,
    });
    const tracer = (instrumentor as unknown as { tracer: ReturnType<typeof makeTracerMock> }).tracer;

    const proto = buildWorkerProto();
    (instrumentor as unknown as { patchWorker: (e: unknown) => void }).patchWorker({
      InternalWorker: { prototype: proto },
    });

    await proto.handleStartStepRun(makeAction());

    const args = getStartStepRunArgs(tracer);
    // 4 args: link path still uses ROOT_CONTEXT
    expect(args).toHaveLength(4);
    expect(args[2]).toBe(MOCK_ROOT_CONTEXT);
    // links array is empty because the span context was invalid
    const opts = args[1] as Record<string, unknown>;
    expect((opts.links as unknown[]).length).toBe(0);
  });
});
