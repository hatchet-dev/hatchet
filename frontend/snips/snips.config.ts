import { Config } from '@/utils/config';

export const config: Config = {
  // Directories to process
  SOURCE_DIRS: ['./test_dir'],

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

    // Python specific
    '__pycache__',
    '.pytest_cache',
    '*.pyc',

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
    },
  ],

  // Patterns to remove from code files
  REMOVAL_PATTERNS: [
    {
      regex: /^\s*(\/\/|#)\s*HH-.*$/gm,
      description: 'HH- style comments',
    },
    {
      regex: /^\s*(\/\/|#)\s*!!\s*$/gm,
      description: 'End marker comments',
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
