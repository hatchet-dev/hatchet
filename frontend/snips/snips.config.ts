import { Config } from '@/utils/config';

export const config: Config = {
  // Directories to process
  SOURCE_DIRS: {
    typescript: '../../sdks/typescript/src/v1/examples',
    python: '../../sdks/python/examples',
    go: '../../pkg/examples',
  },

  // Output directory
  OUTPUT_DIR: 'out',

  // Files to preserve during removal
  PRESERVE_FILES: [
    'package.json',
    'pnpm-lock.yaml',
    'pyproject.toml',
    'README.md',
    'tsconfig.json',
  ],

  // Files and directories to ignore during copying
  IGNORE_LIST: [
    // Test files and directories
    'test',
    'tests',
    '__tests__',
    '*.test.*',
    '*.spec.*',
    '*.test-d.*',

    'package-lock.json',
    'pnpm-lock.yaml',
    'package.json',

    // Python specific
    '__pycache__',
    '.pytest_cache',
    '*.pyc',

    // Go specific
    'go.mod',
    'go.sum',
    '*_test.go',

    // System files
    '.DS_Store',

    // Development directories
    'node_modules',
    '.git',
    '*.log',
    '*.tmp',
    '.env',
    '.venv',
    'venv',
    'dist',
    'build',
  ],

  // Text replacements to perform on copied files
  REPLACEMENTS: [
    {
      from: '@hatchet',
      to: '@hatchet-dev/typescript-sdk',
      fileTypes: ['ts'],
    },
  ],

  // Patterns to remove from code files
  REMOVAL_PATTERNS: [
    {
      regex: '# HH-',
      description: 'HH- style comments',
    },
    {
      regex: '# !!',
      description: 'End marker comments',
    },
    {
      regex: '// !!',
      description: 'End marker comments',
    },
    {
      regex: '// HH-',
      description: 'HH- style comments',
    },
    {
      regex: /^\s*\/\*\s*eslint-.*\*\/$/gm,
      description: 'ESLint disable block comments',
    },
    {
      regex: /\s*(\/\/|#)\s*eslint-disable-next-line.*$/gm,
      description: 'ESLint disable line comments',
    },
  ],
};
