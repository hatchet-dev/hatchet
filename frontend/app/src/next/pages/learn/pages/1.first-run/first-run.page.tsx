import { MembersProvider } from '@/next/hooks/use-members';
import { TwoColumnLayout } from '@/next/components/layouts/two-column.layout';
import { GithubCode } from '@/next/components/ui/code/github-code';
import { useMemo, useState, useRef, useEffect } from 'react';
import {
  lessonPlan,
  FirstRunStepKeys,
  FirstRunExtra,
} from './first-run.lesson-plan';
import { HighlightStateMap } from '../../components/lesson-plan';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';

export default function OnboardingFirstRunPage() {
  return (
    <MembersProvider>
      <OnboardingFirstRunContent />
    </MembersProvider>
  );
}

function OnboardingFirstRunContent() {
  const isMobile = useIsMobile();

  const [language, setLanguage] = useState(lessonPlan.defaultLanguage);

  const [activeStep, setActiveStep] = useState<FirstRunStepKeys>('intro');

  const [extra, setExtra] = useState(lessonPlan.extraDefaults);

  const [highlightState, setHighlightState] = useState<
    HighlightStateMap<FirstRunStepKeys>
  >({});

  const codeBlocksRef = useRef<Record<FirstRunStepKeys, HTMLDivElement | null>>(
    {
      intro: null,
      setup: null,
      task: null,
      worker: null,
      run: null,
    },
  );

  const stepCardsRef = useRef<Record<FirstRunStepKeys, HTMLDivElement | null>>({
    intro: null,
    setup: null,
    task: null,
    worker: null,
    run: null,
  });

  useEffect(() => {
    const activeBlock = codeBlocksRef.current[activeStep];
    const activeCard = stepCardsRef.current[activeStep];
    if (activeBlock) {
      activeBlock.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
    if (activeCard) {
      activeCard.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [activeStep]);

  const commands = useMemo(() => {
    const languageExtra = extra[language];
    const defaultExtra = lessonPlan.extraDefaults[language];

    if (!languageExtra || !defaultExtra) {
      throw new Error(`Invalid language: ${language}`);
    }

    const packageManager =
      languageExtra.packageManager || defaultExtra.packageManager;
    if (!packageManager) {
      throw new Error(`Invalid package manager for language: ${language}`);
    }

    return lessonPlan.commands[packageManager];
  }, [language, extra]);

  const props = useMemo(
    () => ({
      highlights: highlightState,
      setHighlights: (highlightState: HighlightStateMap<FirstRunStepKeys>) =>
        setHighlightState(highlightState),
      activeStep,
      setActiveStep,
      extra: extra[language],
      setExtra: (e: Partial<FirstRunExtra>) =>
        setExtra((prev) => ({
          ...prev,
          [language]: { ...prev[language], ...e },
        })),
      language,
      setLanguage,
      commands,
    }),
    [
      commands,
      highlightState,
      setHighlightState,
      activeStep,
      setActiveStep,
      extra,
      setExtra,
      language,
      setLanguage,
    ],
  );

  const stepKeys = Object.keys(lessonPlan.steps) as FirstRunStepKeys[];
  const currentStepIndex = stepKeys.indexOf(activeStep);

  return (
    <TwoColumnLayout
      left={
        <div className="space-y-4 pb-[600px]">
          {Object.entries(lessonPlan.steps).map(([key, step]) => {
            const isActive = key === activeStep;
            const stepIndex = stepKeys.indexOf(key as FirstRunStepKeys);
            const isCompleted = stepIndex < currentStepIndex;

            return (
              <div
                key={key}
                ref={(el) =>
                  (stepCardsRef.current[key as FirstRunStepKeys] = el)
                }
                className={cn('transition-opacity duration-200', {
                  'opacity-100': isActive || isCompleted,
                  'opacity-50': !isActive && !isCompleted,
                })}
              >
                {step.description(props as any)}
                {isActive && currentStepIndex < stepKeys.length - 1 && (
                  <div className="mt-4 flex justify-end gap-2 items-center">
                    <Button
                      variant="outline"
                      onClick={() => {
                        setActiveStep(stepKeys[currentStepIndex - 1]);
                      }}
                    >
                      Skip Tutorial
                    </Button>
                    <Button
                      onClick={() => {
                        const nextStep = stepKeys[currentStepIndex + 1];
                        setActiveStep(nextStep);
                      }}
                    >
                      Continue
                    </Button>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      }
      right={
        <>
          <div className="space-y-2 pb-[600px]">
            {Object.entries(lessonPlan.steps).map(([key, step]) => {
              const languageCode = step.githubCode?.[language];
              if (!languageCode) {
                return null;
              }

              const stepIndex = stepKeys.indexOf(key as FirstRunStepKeys);

              return (
                <div
                  key={key}
                  ref={(el) =>
                    (codeBlocksRef.current[key as FirstRunStepKeys] = el)
                  }
                  className={cn('transition-opacity duration-200', {
                    'opacity-100': stepIndex <= currentStepIndex,
                    'opacity-50': stepIndex > currentStepIndex,
                  })}
                >
                  {step.content && <div>{step.content(props as any)}</div>}
                  {languageCode && (
                    <GithubCode
                      className={cn({
                        'opacity-30': stepIndex > currentStepIndex,
                      })}
                      key={key}
                      highlightLines={
                        highlightState[key as FirstRunStepKeys]?.lines
                      }
                      highlightStrings={
                        highlightState[key as FirstRunStepKeys]?.strings
                      }
                      {...lessonPlan.codeBlockDefaults}
                      language={language}
                      repo={
                        (typeof languageCode === 'string'
                          ? lessonPlan.codeBlockDefaults.repos[language]
                          : languageCode.repo) || ''
                      }
                      path={
                        typeof languageCode === 'string'
                          ? languageCode
                          : languageCode.path
                      }
                    />
                  )}
                </div>
              );
            })}
          </div>
        </>
      }
    />
  );
}
