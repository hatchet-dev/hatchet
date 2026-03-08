import eslint from '@eslint/js';
import eslintConfigPrettier from 'eslint-config-prettier';
import eslintPluginPrettier from 'eslint-plugin-prettier';
import unusedImports from 'eslint-plugin-unused-imports';
import tseslint from 'typescript-eslint';

export default [
  {
    ignores: ['dist/**', '**/generated/**/*', 'node_modules/**', '**/*.test-d.ts'],
  },

  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  eslintConfigPrettier,

  {
    plugins: {
      prettier: eslintPluginPrettier,
      'unused-imports': unusedImports,
    },
    rules: {
      'prettier/prettier': 'error',
      'unused-imports/no-unused-imports': 'error',

      'no-void': 'off',
      'no-use-before-define': 'off',
      'no-await-in-loop': 'off',
      'no-restricted-syntax': 'off',
      curly: 'error',
      'prefer-destructuring': ['error', { object: true, array: false }],

      '@typescript-eslint/no-shadow': 'off',
      '@typescript-eslint/no-use-before-define': 'off',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-empty-object-type': 'off',
      '@typescript-eslint/no-require-imports': 'off',

      'class-methods-use-this': 'off',
    },
  },

  {
    files: [
      'src/**/examples/**/*.{ts,tsx,js}',
      'src/examples/**/*.{ts,tsx,js}',
      'tests/**/*.{ts,tsx,js}',
      'src/**/*.test.{ts,tsx,js}',
      'src/**/*.e2e.{ts,tsx,js}',
      'src/**/__tests__/**/*.{ts,tsx,js}',
    ],
    rules: {
      '@typescript-eslint/no-unused-vars': 'off',
      'no-console': 'off',
    },
  },
];
