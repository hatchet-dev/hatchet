// Option catalogs for the Overview onboarding. These live outside the
// component file so the pure catalog, command, and persistence modules can
// import them without pulling React or UI imports into unit tests.

export const workflowStepOptions = {
  chooseUseCase: { value: 'chooseUseCase', label: 'Choose use case' },
  install: { value: 'install', label: 'Install the CLI' },
  profile: { value: 'profile', label: 'Set your profile' },
  quickstart: { value: 'quickstart', label: 'Project quickstart' },
  runTask: { value: 'runTask', label: 'Run a task' },
  // The key predates the label. This tab used to be the Docs MCP step,
  // and persisted tab state may still reference the old key.
  aiDocs: { value: 'aiDocs', label: 'Finish' },
} as const;

export const workflowLanguageOptions = {
  python: { value: 'python', label: 'Python' },
  typescript: { value: 'typescript', label: 'TypeScript' },
  go: { value: 'go', label: 'Go' },
} as const;

export const installMethodOptions = {
  native: { value: 'native', label: 'Native (Recommended)' },
  homebrew: { value: 'homebrew', label: 'Homebrew' },
} as const;

export type WorkflowStepKey = keyof typeof workflowStepOptions;
export type WorkflowLanguageKey =
  (typeof workflowLanguageOptions)[keyof typeof workflowLanguageOptions]['value'];
export type InstallMethod =
  (typeof installMethodOptions)[keyof typeof installMethodOptions]['value'];
