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

  {
    files: [
      'src/v1/types.ts',
      'src/v1/declaration.ts',
      'src/v1/client/client.ts',
      'src/v1/task.ts',
      'src/clients/hatchet-client/client-config.ts',
      'src/legacy/step.ts',
      'src/legacy/workflow.ts',
      'src/legacy/legacy-transformer.ts',
      'src/v1/client/worker/worker-internal.ts',
      'src/v1/client/worker/context.ts',
      'src/v1/client/worker/slot-utils.ts',
      'src/v1/client/admin.ts',
      'src/v1/client/worker/worker.ts',
      'src/v1/client/features/workflows.ts',
      'src/v1/client/features/crons.ts',
      'src/v1/client/features/runs.ts',
      'src/v1/client/features/cel.ts',
      'src/v1/client/features/schedules.ts',
      'src/v1/conditions/parent-condition.ts',
      'src/v1/conditions/transformer.ts',
    ],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },

  {
    files: [
      'src/**/*.test.{ts,tsx}',
      'src/**/*.e2e.{ts,tsx}',
      'tests/**/*.{ts,tsx}',
      'src/**/examples/**/*.{ts,tsx,js}',
      'src/**/__e2e__/**/*.{ts,tsx}',
    ],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
];
