'use strict';

const path = require('path');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');

/** @type {import('webpack').Configuration[]} */
module.exports = [
  // ─── Target 1: Extension host (Node.js) ───────────────────────────────────
  {
    name: 'extension',
    target: 'node',
    mode: 'none',
    entry: './src/extension.ts',
    output: {
      path: path.resolve(__dirname, 'dist'),
      filename: 'extension.js',
      libraryTarget: 'commonjs2',
    },
    resolve: {
      extensions: ['.ts', '.js'],
      alias: {
        // The extension host only uses `import type` from the shared package,
        // so this module will never appear in the compiled output. The alias
        // is here as a safety net so webpack can resolve it if needed.
        '@hatchet-dev/dag-visualizer': path.resolve(
          __dirname,
          'src/dag-visualizer/index.ts',
        ),
      },
    },
    module: {
      rules: [
        {
          test: /\.ts$/,
          exclude: /node_modules/,
          use: {
            loader: 'ts-loader',
            options: {
              configFile: path.resolve(__dirname, 'tsconfig.json'),
              transpileOnly: true,
            },
          },
        },
      ],
    },
    // The vscode module is provided by VS Code at runtime — never bundle it.
    externals: {
      vscode: 'commonjs vscode',
    },
    devtool: 'nosources-source-map',
    infrastructureLogging: { level: 'log' },
  },

  // ─── Target 2: Webview React app (browser) ────────────────────────────────
  {
    name: 'webview',
    target: 'web',
    mode: 'none',
    entry: './src/webview/index.tsx',
    output: {
      path: path.resolve(__dirname, 'dist'),
      filename: 'webview.js',
    },
    resolve: {
      extensions: ['.ts', '.tsx', '.js', '.jsx'],
      alias: {
        '@hatchet-dev/dag-visualizer': path.resolve(
          __dirname,
          'src/dag-visualizer/index.ts',
        ),
      },
    },
    module: {
      rules: [
        // TypeScript / TSX
        {
          test: /\.tsx?$/,
          exclude: /node_modules/,
          use: {
            loader: 'ts-loader',
            options: {
              configFile: path.resolve(__dirname, 'tsconfig.webview.json'),
              transpileOnly: true,
            },
          },
        },
        // CSS — extract to webview.css (included via <link> in the webview HTML)
        {
          test: /\.css$/,
          use: [
            MiniCssExtractPlugin.loader,
            'css-loader',
            {
              loader: 'postcss-loader',
              options: {
                postcssOptions: {
                  config: path.resolve(__dirname, 'postcss.config.js'),
                },
              },
            },
          ],
        },
      ],
    },
    plugins: [
      new MiniCssExtractPlugin({ filename: 'webview.css' }),
    ],
    devtool: 'nosources-source-map',
  },
];
