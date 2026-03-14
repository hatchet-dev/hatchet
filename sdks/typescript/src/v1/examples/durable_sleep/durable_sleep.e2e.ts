import { durableSleep } from './workflow';

describe('durable-sleep-e2e', () => {
  it('sleeps for 5s and completes', async () => {
    const start = Date.now();
    const result = await durableSleep.run({});
    const elapsed = (Date.now() - start) / 1000;

    expect(result).toEqual(
      expect.objectContaining({
        'durable-sleep': expect.objectContaining({
          Value: 'done',
        }),
      })
    );
    expect(elapsed).toBeGreaterThanOrEqual(4);
    expect(elapsed).toBeLessThanOrEqual(30);
  }, 120_000);
});
