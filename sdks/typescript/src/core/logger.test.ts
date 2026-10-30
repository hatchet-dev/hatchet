import { HatchetCore } from './client';
import { ConsoleLogger } from './logger';

const TOKEN = `eyJhbGciOi.${Buffer.from(JSON.stringify({ sub: 'tenant' })).toString('base64url')}.sig`;
const METHODS = ['debug', 'info', 'warn', 'error'] as const;

function spyConsole() {
  const spies = METHODS.map((method) => jest.spyOn(console, method).mockImplementation(() => {}));
  return {
    calls: () => spies.flatMap((spy) => spy.mock.calls.map((args) => String(args[0]))),
    restore: () => spies.forEach((spy) => spy.mockRestore()),
  };
}

describe('ConsoleLogger', () => {
  it('writes nothing at any level when the level is OFF', () => {
    const output = spyConsole();
    try {
      const logger = new ConsoleLogger('test', 'OFF');
      logger.debug('debug');
      logger.info('info');
      logger.green('green');
      logger.warn('warn', new Error('boom'));
      logger.error('error', new Error('boom'));
      expect(output.calls()).toEqual([]);
    } finally {
      output.restore();
    }
  });

  it('writes from the configured level up, INFO by default', () => {
    const output = spyConsole();
    try {
      const logger = new ConsoleLogger('test');
      logger.debug('hidden');
      logger.info('shown');
      logger.error('failed', new Error('boom'));
      expect(output.calls()).toEqual(['[INFO/test] shown', '[ERROR/test] failed Error: boom']);

      new ConsoleLogger('test', 'ERROR').warn('hidden');
      expect(output.calls()).toHaveLength(2);
    } finally {
      output.restore();
    }
  });

  it('keeps a client with logLevel OFF silent through a pushed and a failed event', async () => {
    const output = spyConsole();
    try {
      const quiet = new HatchetCore({
        token: TOKEN,
        serverUrl: 'https://engine.example',
        logLevel: 'OFF',
        retrier: { maxAttempts: 1 },
        fetch: async () =>
          new Response(new Uint8Array(), { headers: { 'content-type': 'application/proto' } }),
      });
      await quiet.events.push('event', { field: 'value' });

      const failing = new HatchetCore({
        token: TOKEN,
        serverUrl: 'https://engine.example',
        logLevel: 'OFF',
        retrier: { maxAttempts: 1 },
        fetch: async () => {
          throw new Error('remote error');
        },
      });
      await expect(failing.events.push('event', {})).rejects.toThrow();

      expect(output.calls()).toEqual([]);
    } finally {
      output.restore();
    }
  });
});
