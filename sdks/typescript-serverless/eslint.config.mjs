import eslint from '@eslint/js';
import eslintConfigPrettier from 'eslint-config-prettier';
import eslintPluginPrettier from 'eslint-plugin-prettier';
import unusedImports from 'eslint-plugin-unused-imports';
import tseslint from 'typescript-eslint';

// Mirrors sdks/typescript/eslint.config.mjs.
export default [
  {
    ignores: ['dist/**', '**/generated/**/*', 'node_modules/**'],
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
      'prefer-destructuring': 'error',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-empty-object-type': 'off',
    },
  },

  {
    files: ['scripts/**/*.mjs'],
    languageOptions: {
      globals: { process: 'readonly', console: 'readonly' },
    },
  },

  {
    files: ['src/**/*.test.ts'],
    rules: {
      '@typescript-eslint/no-unused-vars': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      'no-console': 'off',
    },
  },

  {
    // Declarations are generic over any input and output; the registry and the
    // testing helper accept every one of them.
    files: ['src/handler/registry.ts', 'src/testing/index.ts', 'src/index.ts'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
];
