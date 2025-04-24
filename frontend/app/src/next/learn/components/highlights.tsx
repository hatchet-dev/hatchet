import { SupportedLanguage } from './lesson-plan';

export type Highlights<
  S extends string,
  C extends string | number | symbol,
> = Record<SupportedLanguage, Record<S, Partial<Record<C, HighlightState>>>>;

export type HighlightState = {
  lines?: number[];
  strings?: string[];
};

export type HighlightStateMap<S extends string> = Partial<
  Record<S, HighlightState>
>;

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
