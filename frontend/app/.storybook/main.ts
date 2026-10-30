import type { StorybookConfig } from '@storybook/react-vite';

const excludedPlugins = ['sentry', 'inject-go-template'];

const isExcluded = (name: string) =>
  excludedPlugins.some((prefix) => name.startsWith(prefix));

const config: StorybookConfig = {
  stories: ['../src/**/*.mdx', '../src/**/*.stories.@(ts|tsx)'],
  addons: ['@storybook/addon-docs', '@storybook/addon-a11y'],
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
