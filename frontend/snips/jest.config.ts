import type { Config } from 'jest';

const config: Config = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/src'],
  transform: {
    '^.+\\.tsx?$': 'ts-jest',
  },
  moduleNameMapper: {
    '^snips.config$': '<rootDir>/snips.config.ts',
    '^@/(.*)$': '<rootDir>/src/$1',
  },
};

export default config;
