import { SupportedLanguage } from './lesson-plan';
import { useLesson } from '../hooks/use-lesson';
import { useCallback } from 'react';
import { cn } from '@/next/lib/utils';

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

export function Highlight<F extends string>({
  frame,
  children,
}: {
  frame: F;
  children: React.ReactNode;
}) {
  const { setActiveStep, codeKeyFrames, setHighlights } = useLesson();

  const setActive = useCallback(() => {
    // Get the first step key from the frame's keyframe mapping
    const stepKey = Object.keys(codeKeyFrames[frame])[0];
    if (stepKey) {
      setActiveStep(stepKey);
    }
  }, [setActiveStep, codeKeyFrames, frame]);

  if (!codeKeyFrames?.[frame]) {
    return children;
  }

  return (
    <span
      className={cn(
        'cursor-help',
        'underline underline-offset-4 decoration-dotted decoration-2 decoration-muted-foreground decoration-offset-4',
      )}
      onClick={() => {
        setActive();
      }}
      onMouseEnter={() => {
        setHighlights(codeKeyFrames[frame]);
        // setHasHovered(true);
      }}
      onMouseLeave={() => {
        setHighlights({});
      }}
    >
      {children}
    </span>
  );
}
