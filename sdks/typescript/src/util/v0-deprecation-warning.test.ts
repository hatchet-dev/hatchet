import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { warnLegacyWorkflow } from '@hatchet/legacy/legacy-transformer';
import { AdminClient } from '@clients/admin/admin-client';
import { createChannel, createClientFactory } from 'nice-grpc';
import {
  V0_DEPRECATION_CODE,
  _resetEmittedV0Warnings,
  emitV0RemovedWarning,
} from './v0-deprecation-warning';

describe('emitV0RemovedWarning', () => {
  let emitWarningSpy: jest.SpyInstance;

  beforeEach(() => {
    _resetEmittedV0Warnings();
    emitWarningSpy = jest.spyOn(process, 'emitWarning').mockImplementation(() => {});
  });

  afterEach(() => {
    emitWarningSpy.mockRestore();
  });

  it('emits via process.emitWarning with the stable HATCHET_V0_REMOVED code', () => {
    emitV0RemovedWarning('workflow');

    expect(emitWarningSpy).toHaveBeenCalledTimes(1);
    const [[message, opts]] = emitWarningSpy.mock.calls;
    expect(message).toContain('workflow module');
    expect(message).toContain('v1.14.0');
    expect(message).toContain('https://docs.hatchet.run/home/v1-sdk-improvements');
    expect(opts).toMatchObject({
      type: 'DeprecationWarning',
      code: V0_DEPRECATION_CODE,
    });
  });

  it('passes the optional detail string through to process.emitWarning', () => {
    emitV0RemovedWarning('workflow', 'ConcurrencyLimitStrategy has moved.');

    expect(emitWarningSpy).toHaveBeenCalledTimes(1);
    const [[, opts]] = emitWarningSpy.mock.calls;
    expect(opts.detail).toBe('ConcurrencyLimitStrategy has moved.');
  });

  it('deduplicates per submodule across repeated calls', () => {
    emitV0RemovedWarning('workflow');
    emitV0RemovedWarning('workflow');
    emitV0RemovedWarning('workflow');

    expect(emitWarningSpy).toHaveBeenCalledTimes(1);
  });

  it('emits separate warnings for different submodules', () => {
    emitV0RemovedWarning('workflow');
    emitV0RemovedWarning('step');

    expect(emitWarningSpy).toHaveBeenCalledTimes(2);
    expect(emitWarningSpy.mock.calls[0][0]).toContain('workflow module');
    expect(emitWarningSpy.mock.calls[1][0]).toContain('step module');
  });

  it('routes to console.warn when process.throwDeprecation is set, bypassing emitWarning', () => {
    // Under --throw-deprecation or process.throwDeprecation=true, Node queues
    // `throw warning` on the next tick AFTER emitWarning returns, so a
    // try/catch around the call cannot intercept it. The helper must check
    // the flag up front and avoid calling emitWarning at all.
    const originalThrowDeprecation = process.throwDeprecation;
    const consoleWarnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    process.throwDeprecation = true;

    try {
      expect(() => emitV0RemovedWarning('workflow')).not.toThrow();
      expect(emitWarningSpy).not.toHaveBeenCalled();
      expect(consoleWarnSpy).toHaveBeenCalledTimes(1);
      expect(consoleWarnSpy.mock.calls[0][0]).toContain(V0_DEPRECATION_CODE);
      expect(consoleWarnSpy.mock.calls[0][0]).toContain('workflow module');
    } finally {
      process.throwDeprecation = originalThrowDeprecation;
      consoleWarnSpy.mockRestore();
    }
  });

  it('falls back to console.warn if emitWarning throws synchronously (non-Node hosts)', () => {
    // Defense-in-depth: a polyfilled `process.emitWarning` in a non-Node
    // host could throw synchronously. Real Node throws asynchronously under
    // --throw-deprecation; that path is exercised by the test above.
    emitWarningSpy.mockImplementation(() => {
      throw new Error('synchronous emitWarning failure');
    });
    const consoleWarnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});

    try {
      expect(() => emitV0RemovedWarning('workflow')).not.toThrow();
      expect(emitWarningSpy).toHaveBeenCalledTimes(1);
      expect(consoleWarnSpy).toHaveBeenCalledTimes(1);
      expect(consoleWarnSpy.mock.calls[0][0]).toContain(V0_DEPRECATION_CODE);
    } finally {
      consoleWarnSpy.mockRestore();
    }
  });

  it('falls back to console.warn when process.emitWarning is unavailable', () => {
    emitWarningSpy.mockRestore();
    const original = process.emitWarning;
    // Simulate a runtime that doesn't expose emitWarning.
    (process as unknown as { emitWarning?: typeof process.emitWarning }).emitWarning =
      undefined as unknown as typeof process.emitWarning;
    const consoleWarnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});

    try {
      emitV0RemovedWarning('step');

      expect(consoleWarnSpy).toHaveBeenCalledTimes(1);
      expect(consoleWarnSpy.mock.calls[0][0]).toContain(V0_DEPRECATION_CODE);
      expect(consoleWarnSpy.mock.calls[0][0]).toContain('step module');
    } finally {
      consoleWarnSpy.mockRestore();
      (process as unknown as { emitWarning: typeof process.emitWarning }).emitWarning = original;
    }
  });
});

describe('importing the SDK', () => {
  it('emits no deprecation warning from the root or v1 entries', () => {
    const emitWarning = jest.spyOn(process, 'emitWarning').mockImplementation(() => {});
    const consoleWarn = jest.spyOn(console, 'warn').mockImplementation(() => {});

    // isolateModules needs require so the modules evaluate fresh inside the isolated registry
    /* eslint-disable @typescript-eslint/no-require-imports */
    jest.isolateModules(() => {
      require('../index');
      require('../v1');
      require('../workflow');
      require('../step');
    });
    /* eslint-enable @typescript-eslint/no-require-imports */

    expect(emitWarning).not.toHaveBeenCalled();
    expect(consoleWarn).not.toHaveBeenCalled();

    emitWarning.mockRestore();
    consoleWarn.mockRestore();
  });
});

describe('using a v0 workflow', () => {
  it('warns once, with the banner and the coded deprecation', () => {
    _resetEmittedV0Warnings();
    const emitWarning = jest.spyOn(process, 'emitWarning').mockImplementation(() => {});
    const consoleWarn = jest.spyOn(console, 'warn').mockImplementation(() => {});

    warnLegacyWorkflow();
    warnLegacyWorkflow();

    expect(consoleWarn).toHaveBeenCalledTimes(1);
    expect(emitWarning).toHaveBeenCalledTimes(1);
    expect(emitWarning.mock.calls[0][1]).toMatchObject({ code: V0_DEPRECATION_CODE });

    emitWarning.mockRestore();
    consoleWarn.mockRestore();
  });

  it('warns when a v0 definition is put to the engine directly', async () => {
    _resetEmittedV0Warnings();
    const emitWarning = jest.spyOn(process, 'emitWarning').mockImplementation(() => {});

    const mockChannel = createChannel('localhost:50051');
    const mockFactory = createClientFactory();
    const admin = new AdminClient(
      {
        token: 't',
        host_port: 'h',
        tls_config: { tls_strategy: 'none' },
        logger: DEFAULT_LOGGER,
      } as any,
      mockChannel,
      mockFactory,
      {} as any,
      'tenantId',
      {} as any,
      {} as any
    );
    (admin as any).client = { putWorkflow: jest.fn().mockResolvedValue({}) };

    await admin.putWorkflow({ name: 'v0', steps: [] } as any);

    expect(emitWarning).toHaveBeenCalledTimes(1);
    expect(emitWarning.mock.calls[0][1]).toMatchObject({ code: V0_DEPRECATION_CODE });

    emitWarning.mockRestore();
  });
});
