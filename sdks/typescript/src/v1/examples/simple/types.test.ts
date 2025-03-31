// eslint-disable-next-line import/no-extraneous-dependencies
import { expectTypeTestsToPassAsync } from 'jest-tsd';

describe('test-typing', () => {
  it('should not produce static type errors', async () => {
    await expectTypeTestsToPassAsync(__filename);
  });
});
