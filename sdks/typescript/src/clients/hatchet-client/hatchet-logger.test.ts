import { HatchetLogger } from './hatchet-logger';

describe('HatchetLogger', () => {
  const spies = () => ({
    log: jest.spyOn(console, 'log').mockImplementation(() => {}),
    warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
    error: jest.spyOn(console, 'error').mockImplementation(() => {}),
    debug: jest.spyOn(console, 'debug').mockImplementation(() => {}),
    info: jest.spyOn(console, 'info').mockImplementation(() => {}),
  });

  afterEach(() => jest.restoreAllMocks());

  it('writes nothing at OFF', async () => {
    const s = spies();
    const logger = new HatchetLogger('test', 'OFF');

    await logger.debug('d');
    await logger.info('i');
    await logger.green('g');
    await logger.warn('w');
    await logger.error('e');

    for (const spy of Object.values(s)) expect(spy).not.toHaveBeenCalled();
  });

  it('writes at and above the configured level', async () => {
    const s = spies();
    const logger = new HatchetLogger('test', 'WARN');

    await logger.info('i');
    await logger.warn('w');
    await logger.error('e');

    const calls = Object.values(s).reduce((n, spy) => n + spy.mock.calls.length, 0);
    expect(calls).toBe(2);
  });
});
