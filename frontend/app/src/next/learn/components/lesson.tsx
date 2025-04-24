import { TwoColumnLayout } from '@/next/components/layouts/two-column.layout';
import { GithubCode } from '@/next/components/ui/code/github-code';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { LessonProvider, useLesson } from '@/next/learn/hooks/use-lesson';
import {
  LessonPlan,
  SupportedLanguage,
  LessonStep,
} from '@/next/learn/components/lesson-plan';

export function Lesson<
  S extends string,
  E extends Record<string, unknown> = Record<string, unknown>,
  C = Record<string, string>,
>({ lesson }: { lesson: LessonPlan<S, E, C> }) {
  return (
    <LessonProvider lesson={lesson}>
      <LessonContent lesson={lesson} />
    </LessonProvider>
  );
}

function LessonContent<
  S extends string,
  E extends Record<string, unknown> = Record<string, unknown>,
  C = Record<string, string>,
>({ lesson }: { lesson: LessonPlan<S, E, C> }) {
  const {
    language,
    activeStep,
    setActiveStep,
    highlights,
    stepKeys,
    currentStepIndex,
    codeBlocksRef,
    stepCardsRef,
  } = useLesson();

  return (
    <TwoColumnLayout
      left={
        <div className="space-y-8 pb-[600px]">
          {Object.entries(lesson.steps).map(([key, step]) => {
            const typedStep = step as LessonStep;
            const isActive = key === activeStep;
            const stepIndex = stepKeys.indexOf(key as S);
            const isCompleted = stepIndex < currentStepIndex;

            return (
              <div
                key={key}
                ref={(el) => (stepCardsRef.current[key as S] = el)}
                className={cn('transition-opacity duration-200', {
                  'opacity-100': isActive || isCompleted,
                  'opacity-50': !isActive && !isCompleted,
                })}
              >
                {typedStep.description()}
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
          <div className="space-y-8 pb-[600px]">
            {Object.entries(lesson.steps).map(([key, step]) => {
              const typedStep = step as LessonStep;
              const languageCode =
                typedStep.githubCode?.[language as SupportedLanguage];
              if (!languageCode) {
                return null;
              }

              const stepIndex = stepKeys.indexOf(key as S);

              return (
                <div
                  key={key}
                  ref={(el) => (codeBlocksRef.current[key as S] = el)}
                  className={cn('transition-opacity duration-200', {
                    'opacity-100': stepIndex <= currentStepIndex,
                    'opacity-50': stepIndex > currentStepIndex,
                  })}
                >
                  {typedStep.content && <div>{typedStep.content()}</div>}
                  {languageCode && (
                    <GithubCode
                      className={cn({
                        'opacity-30': stepIndex > currentStepIndex,
                      })}
                      key={key}
                      highlightLines={highlights[key as S]?.lines}
                      highlightStrings={highlights[key as S]?.strings}
                      {...lesson.codeBlockDefaults}
                      language={language}
                      repo={
                        (typeof languageCode === 'string'
                          ? lesson.codeBlockDefaults.repos[
                              language as SupportedLanguage
                            ]
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
