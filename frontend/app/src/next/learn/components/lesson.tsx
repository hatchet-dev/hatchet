import { GithubCode } from '@/next/components/ui/code/github-code';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { LessonProvider, useLesson } from '@/next/learn/hooks/use-lesson';
import { Card, CardContent } from '@/next/components/ui/card';
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

  const steps = Object.entries(lesson.steps).map(([key, step], i) => {
    const typedStep = step as LessonStep;
    const isActive = key === activeStep;
    const stepIndex = stepKeys.indexOf(key as S);
    const isCompleted = stepIndex < currentStepIndex;
    const languageCode = typedStep.githubCode?.[language as SupportedLanguage];

    return (
      <div key={key} className="flex gap-16">
        <div
          className="w-[38.2%]"
          ref={(el) => (stepCardsRef.current[key as S] = el)}
        >
          <div
            className={cn('transition-opacity duration-200', {
              'opacity-100': isActive || isCompleted,
              'opacity-50': !isActive && !isCompleted,
            })}
          >
            {typedStep.description()}
            {stepIndex !== stepKeys.length - 1 && (
              <div className="mt-4 flex justify-end gap-2 items-center">
                {i == 0 && (
                  <Button
                    variant="outline"
                    onClick={() => {
                      setActiveStep(stepKeys[currentStepIndex - 1]);
                    }}
                  >
                    Skip Tutorial
                  </Button>
                )}
                <Button
                  variant={isActive ? 'default' : 'outline'}
                  onClick={() => {
                    setActiveStep(stepKeys[i + 1]);
                  }}
                >
                  {currentStepIndex == 0 ? 'Get Started' : 'Continue'}
                </Button>
              </div>
            )}
          </div>
        </div>

        {languageCode && (
          <div
            className="w-[61.8%] pt-16"
            ref={(el) => (codeBlocksRef.current[key as S] = el)}
          >
            <div
              className={cn('transition-opacity duration-200', {
                'opacity-100': stepIndex <= currentStepIndex,
                'opacity-50': stepIndex > currentStepIndex,
              })}
            >
              <Card>
                <CardContent className="space-y-6 py-4">
                  {typedStep.content && <div>{typedStep.content()}</div>}
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
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </div>
    );
  });

  return (
    <div className="h-[calc(100vh-4rem)] overflow-y-auto p-4">
      <div className="mx-auto max-w-7xl pt-48">
        <div className="flex flex-col gap-48 pb-48">{steps}</div>
      </div>
    </div>
  );
}
