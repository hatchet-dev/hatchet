/* eslint-disable no-param-reassign */

import type { NextConfig } from 'next';
import path from 'path';

const nextConfig: NextConfig = {
  webpack: (config) => {
    config.resolve.alias = {
      ...config.resolve.alias,
      '@hatchet/v1': path.resolve(__dirname, '../../../../../../src/v1'),
      '@hatchet-dev/typescript-sdk': path.resolve(__dirname, '../../../../../../src'),
    };
    return config;
  },
};

export default nextConfig;
