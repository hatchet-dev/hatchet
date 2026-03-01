/**
 * Jest config for e2e tests - uses shared worker via globalSetup/globalTeardown.
 */
import type { Config } from 'jest';
import baseConfig from './jest.config';

const config: Config = {
  ...baseConfig,
  testMatch: ['**/*.e2e.ts'],
  globalSetup: '<rootDir>/jest.e2e-global-setup.ts',
  globalTeardown: '<rootDir>/jest.e2e-global-teardown.ts',
};

export default config;
