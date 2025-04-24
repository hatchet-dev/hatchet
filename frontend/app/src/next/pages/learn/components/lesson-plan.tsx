import { GithubCode } from '@/next/components/ui/code/github-code';

export interface LessonPlan<S extends string, E> {
  title: string;
  description: React.ReactNode;
  extraDefaults: E;
  steps: Record<S, LessonStep>;
  codeBlockDefaults: {
    repo: string;
    language: string;
    showLineNumbers: boolean;
  };
}

export interface LessonStep {
  title: string;
  description: (props: LessonPlanStepProps<any, any>) => React.ReactNode;
  content?: (props: LessonPlanStepProps<any, any>) => React.ReactNode;
  githubCode?: Partial<typeof GithubCode> & { path: string };
}

export type HighlightState = {
  lines?: number[];
  strings?: string[];
};

export type HighlightStateMap<S extends string> = Partial<
  Record<S, HighlightState>
>;

export interface LessonPlanStepProps<S extends string, E> {
  highlights: HighlightStateMap<S>;
  setHighlights: (key: S, highlightState?: HighlightState) => void;
  focus?: S;
  setFocus: (key?: S) => void;
  extra: E;
  setExtra: (e: Partial<E>) => void;
}
