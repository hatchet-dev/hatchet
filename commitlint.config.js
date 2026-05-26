module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'build', // Build system or external dependencies
        'chore', // Other changes that don't modify src or test files
        'ci', // CI configuration changes
        'docs', // Documentation only changes
        'feat', // New feature
        'fix', // Bug fix
        'refactor', // Code change that neither fixes a bug nor adds a feature
        'revert', // Revert a previous commit
        'style', // Code style changes (formatting, linting, etc.)
        'test', // Adding or updating tests
        'perf', // Performance improvements
      ],
    ],
    'subject-case': [0], // Level 0 ignores the rule
    'scope-empty': [2, 'never'],
    'header-max-length': [2, 'always', 120],
  },
};
