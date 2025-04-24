import { MembersProvider } from '@/next/hooks/use-members';
import { TwoColumnLayout } from '@/next/components/layouts/two-column.layout';
import { GithubCode } from '@/next/components/ui/code/github-code';
import { useMemo, useState } from 'react';
import {
  lessonPlan,
  FirstRunStepKeys,
  FirstRunExtra,
} from './first-run.lesson-plan';
import {
  HighlightStateMap,
  HighlightState,
} from '../../components/lesson-plan';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { cn } from '@/next/lib/utils';
export default function OnboardingFirstRunPage() {
  return (
    <MembersProvider>
      <OnboardingFirstRunContent />
    </MembersProvider>
  );
}

function OnboardingFirstRunContent() {
  const isMobile = useIsMobile();

  const [focus, setFocus] = useState<FirstRunStepKeys>('setup');

  const [extra, setExtra] = useState(lessonPlan.extraDefaults);

  const [highlightState, setHighlightState] = useState<
    HighlightStateMap<FirstRunStepKeys>
  >({});

  const props = useMemo(
    () => ({
      highlights: highlightState,
      setHighlights: (key: FirstRunStepKeys, highlightState: HighlightState) =>
        setHighlightState(() => {
          return { [key]: highlightState || {} };
        }),
      focus,
      setFocus,
      extra,
      setExtra: (e: Partial<FirstRunExtra>) =>
        setExtra((prev) => ({ ...prev, ...e })),
    }),
    [highlightState, setHighlightState, focus, setFocus, extra, setExtra],
  );

  return (
    <TwoColumnLayout
      left={
        <div className="space-y-4">
          {Object.entries(lessonPlan.steps).map(([key, step]) => (
            <>{step.description(props as any)}</>
          ))}
        </div>
      }
      right={
        <>
          <div className="space-y-2">
            {Object.entries(lessonPlan.steps).map(([key, step]) => (
              <>
                {step.content && (
                  <div key={key}>{step.content(props as any)}</div>
                )}
                {step.githubCode && (
                  <GithubCode
                    className={cn({
                      'opacity-30': focus !== key,
                    })}
                    key={key}
                    highlightLines={
                      highlightState[key as FirstRunStepKeys]?.lines
                    }
                    highlightStrings={
                      highlightState[key as FirstRunStepKeys]?.strings
                    }
                    {...lessonPlan.codeBlockDefaults}
                    {...step.githubCode}
                  />
                )}
              </>
            ))}
          </div>
        </>
      }
    />
  );
}
