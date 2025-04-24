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

export type Highlights<
  S extends string,
  C extends string | number | symbol,
> = Record<SupportedLanguage, Record<S, Partial<Record<C, HighlightState>>>>;

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

export type HighlightState = {
  lines?: number[];
  strings?: string[];
};

export type HighlightStateMap<S extends string> = Partial<
  Record<S, HighlightState>
>;

export interface LessonPlanStepProps<S extends string, E, C> {
  highlights: HighlightStateMap<S>;
  setHighlights: (highlightState?: HighlightStateMap<S>) => void;
  activeStep?: S;
  setActiveStep: (key?: S) => void;
  extra: E;
  setExtra: (e: Partial<E>) => void;
  language: SupportedLanguage;
  setLanguage: (language: SupportedLanguage) => void;
  commands: C;
}

export type CommandMap<S extends string, T extends Record<string, string>> = {
  [K in S]?: T;
};

export function Highlight<F extends string, S extends string>({
  language,
  frame,
  codeKeyFrames,
  setHighlights,
  children,
}: {
  language: SupportedLanguage;
  frame: F;
  codeKeyFrames: Highlights<F, S>;
  setHighlights: (highlightState?: HighlightStateMap<S>) => void;
  children: React.ReactNode;
}) {
  return (
    <span
      onMouseEnter={() => setHighlights(codeKeyFrames[language][frame])}
      onMouseLeave={() => setHighlights({})}
    >
      {children}
    </span>
  );
}
