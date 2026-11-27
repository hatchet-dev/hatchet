import type { StorybookConfig } from '@storybook/react-vite';

const excludedPlugins = ['sentry', 'inject-go-template'];

const isExcluded = (name: string) =>
  excludedPlugins.some((prefix) => name.startsWith(prefix));

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  // Test-only stories still run in `pnpm test:stories`; the sidebar's tag
  // filter shows them when debugging.
  tags: { 'test-only': { defaultFilterSelection: 'exclude' } },
  addons: [
    '@storybook/addon-docs',
    '@storybook/addon-a11y',
    '@storybook/addon-vitest',
  ],
  // Serves MSW's mockServiceWorker.js. Kept out of the app's public/ so the
  // worker never ships in the production build.
  staticDirs: ['./public'],
  framework: {
    name: '@storybook/react-vite',
    options: {},
  },
  viteFinal: (config) => ({
    ...config,
    plugins: config.plugins
      ?.flat()
      .filter(
        (plugin) => !(plugin && 'name' in plugin && isExcluded(plugin.name)),
      ),
  }),
};

export default config;
