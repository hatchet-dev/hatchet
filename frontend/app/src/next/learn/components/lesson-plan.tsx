import { GithubCode } from '@/next/components/ui/code/github-code';

export type SupportedLanguage = 'typescript' | 'python' | 'go';
export type TypeScriptPackageManager = 'npm' | 'pnpm' | 'yarn';
export type PythonPackageManager = 'poetry' | 'uv' | 'pip' | 'pipenv';
export type GoPackageManager = 'go';

export type PackageManager =
  | TypeScriptPackageManager
  | PythonPackageManager
  | GoPackageManager;

export interface Commands<C> extends Record<PackageManager, C> {}

export interface LanguageSpecificCode {
  path: string;
  repo?: string;
}

export type Extra<E> = Record<SupportedLanguage, Partial<E>>;

export interface LessonPlan<S extends string, E, C> {
  title: string;
  description: React.ReactNode;
  defaultLanguage: SupportedLanguage;
  extraDefaults: Extra<E>;
  steps: Record<S, LessonStep>;
  commands: Commands<C>;
  codeBlockDefaults: {
    showLineNumbers: boolean;
    repos: Record<SupportedLanguage, string>;
  };
}

export interface LessonStep {
  title: string;
  description: () => React.ReactNode;
  content?: () => React.ReactNode;
  githubCode?: Partial<typeof GithubCode> & {
    [K in SupportedLanguage]?: string | LanguageSpecificCode;
  };
}

export type CommandMap<S extends string, T extends Record<string, string>> = {
  [K in S]?: T;
};
