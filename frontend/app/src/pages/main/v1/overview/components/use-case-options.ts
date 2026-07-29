import {
  workflowLanguageOptions,
  type WorkflowLanguageKey,
} from './onboarding-options';

// Selectable quickstart use cases. `value` is the CLI --use-case token and
// `trigger` is the workflow name the generated template registers, so both
// feed directly into the printed commands. Roadmap use cases without a
// template belong in a separate display-only list, never here. The helpers
// below accept only AvailableUseCaseKey, so a display-only entry cannot
// produce a command.
export const availableUseCases = {
  simple: {
    value: 'simple',
    label: 'Simple task',
    description: 'A minimal task that confirms your worker runs end to end.',
    languages: [
      workflowLanguageOptions.python.value,
      workflowLanguageOptions.typescript.value,
      workflowLanguageOptions.go.value,
    ],
    trigger: 'simple',
  },
  scheduled: {
    value: 'scheduled',
    label: 'Scheduled CRON job',
    description:
      'A workflow on a cron schedule that can also be run on demand.',
    languages: [
      workflowLanguageOptions.python.value,
      workflowLanguageOptions.typescript.value,
      workflowLanguageOptions.go.value,
    ],
    trigger: 'manual-run',
  },
} as const;

export type AvailableUseCaseKey = keyof typeof availableUseCases;

export type QuickstartSelection = {
  useCase: AvailableUseCaseKey;
  language: WorkflowLanguageKey;
};

export function escapeForDoubleQuotes(value: string): string {
  return value
    .replace(/\\/g, '\\\\')
    .replace(/"/g, '\\"')
    .replace(/\$/g, '\\$')
    .replace(/`/g, '\\`');
}

// The CLI prints this after scaffolding for every language and package
// manager, so onboarding shows the same command everywhere.
export function workerDevCommand(profile: string): string {
  return `hatchet worker dev --profile "${escapeForDoubleQuotes(profile)}"`;
}

export function isLanguageSupported(
  useCase: AvailableUseCaseKey,
  language: WorkflowLanguageKey,
): boolean {
  return (
    availableUseCases[useCase].languages as readonly WorkflowLanguageKey[]
  ).includes(language);
}

export function resolveLanguage(
  useCase: AvailableUseCaseKey,
  language: WorkflowLanguageKey,
): WorkflowLanguageKey {
  return isLanguageSupported(useCase, language)
    ? language
    : availableUseCases[useCase].languages[0];
}

// The CLI prompts for the package manager, project name, and directory,
// so the command deliberately omits those flags.
export function scaffoldCommand({
  useCase,
  language,
}: QuickstartSelection): string {
  return `hatchet quickstart --use-case ${useCase} --language ${resolveLanguage(
    useCase,
    language,
  )}`;
}

export function triggerCommand(
  useCase: AvailableUseCaseKey,
  profile: string,
): string {
  return `hatchet trigger ${availableUseCases[useCase].trigger} --profile "${escapeForDoubleQuotes(profile)}"`;
}
