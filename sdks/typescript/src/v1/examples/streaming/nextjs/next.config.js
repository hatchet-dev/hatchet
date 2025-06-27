/* eslint-disable no-param-reassign */

import path from 'path';

const nextConfig = {
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
