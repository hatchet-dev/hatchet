/* eslint-disable no-param-reassign,no-underscore-dangle */

import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

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
