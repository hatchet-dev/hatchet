import { GithubCode } from '@/next/components/ui/code/github-code';
import { Button } from '@/next/components/ui/button';
import { LessonProvider, useLesson } from '@/next/learn/hooks/use-lesson';
import { Card, CardContent } from '@/next/components/ui/card';
import {
  LessonPlan,
  SupportedLanguage,
  LessonStep,
} from '@/next/learn/components/lesson-plan';
import { cn } from '@/next/lib/utils';
import { Clock } from 'lucide-react';

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
    const languageCode = typedStep.githubCode?.[language as SupportedLanguage];

    return (
      <div
        key={key}
        className={cn(
          'flex flex-col md:flex-row gap-8 md:gap-12 items-stretch bg-card rounded-xl border p-6 md:p-8 mb-8',
        )}
      >
        <div
          className="md:w-2/5 w-full"
          ref={(el) => (stepCardsRef.current[key as S] = el)}
        >
          <div className="">
            {typedStep.description()}
            {stepIndex !== stepKeys.length - 1 && (
              <div className="mt-4 flex justify-end gap-2 items-center">
                <Button
                  variant={isActive ? 'default' : 'outline'}
                  onClick={() => {
                    setActiveStep(stepKeys[i + 1]);
                  }}
                >
                  Continue
                </Button>
              </div>
            )}
          </div>
        </div>

        {languageCode && (
          <div
            className="md:w-3/5 w-full pt-6 md:pt-0"
            ref={(el) => (codeBlocksRef.current[key as S] = el)}
          >
            <div>
              <Card className="shadow-none border-none bg-transparent">
                <CardContent className="space-y-6 py-0 px-0">
                  {typedStep.content && <div>{typedStep.content()}</div>}
                  <GithubCode
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
    <div
      className="h-[calc(100vh-4rem)] overflow-y-auto p-2 md:p-6"
      onScroll={() => {
        if (activeStep === undefined) {
          setActiveStep(stepKeys[0]);
        }
      }}
    >
      <div className="mx-auto max-w-5xl">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col md:flex-row gap-8 md:gap-12 items-stretch bg-card rounded-xl border p-6 md:p-8 md:my-40">
            <div className="flex flex-col gap-4 w-full">
              {lesson.intro}

              <div className="mt-4 flex justify-between gap-2 items-center">
                <div className="flex gap-2 items-center">
                  <Clock className="w-4 h-4" />
                  <span className="text-sm text-muted-foreground">
                    {lesson.duration}
                  </span>
                </div>

                <div className="flex gap-2 items-center">
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setActiveStep(stepKeys[currentStepIndex - 1]);
                    }}
                  >
                    Skip Tutorial
                  </Button>
                  <Button
                    onClick={() => {
                      setActiveStep(stepKeys[0]);
                    }}
                  >
                    Get Started
                  </Button>
                </div>
              </div>
            </div>
          </div>
          <dl
            className={cn('flex flex-col gap-4', {
              'opacity-30': activeStep === undefined,
            })}
          >
            {steps}
          </dl>
        </div>
      </div>
    </div>
  );
}
