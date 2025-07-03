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
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { Snippet } from '@/next/components/ui/code/snippet';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';

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
    stepKeys,
    currentStepIndex,
    codeBlocksRef,
    stepCardsRef,
  } = useLesson();
  const { tenantId } = useCurrentTenantId();

  const steps = Object.entries(lesson.steps).map(([key, step], stepIndex) => {
    const typedStep = step as LessonStep;
    const isActive = key === activeStep;

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
                    setActiveStep(stepKeys[stepIndex + 1]);
                  }}
                >
                  Continue
                </Button>
              </div>
            )}
          </div>
        </div>

        <div
          className="md:w-3/5 w-full pt-6 md:pt-0"
          ref={(el) => (codeBlocksRef.current[key as S] = el)}
        >
          <div>
            {/* TODO: Add highlight strings */}
            <Card className="shadow-none border-none bg-transparent">
              <CardContent className="space-y-6 py-0 px-0">
                {typedStep.code?.[language as SupportedLanguage] ? (
                  <Snippet
                    // highlightLines={highlights[key as S]?.lines}
                    // highlightStrings={highlights[key as S]?.strings}
                    src={typedStep.code?.[language as SupportedLanguage]}
                  />
                ) : null}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    );
  });

  return (
    <div className="flex flex-col h-full items-center">
      <div
        className="h-[calc(100vh-10rem)] overflow-y-auto p-2 md:p-6"
        onScroll={() => {
          if (activeStep === undefined) {
            setActiveStep(stepKeys[0]);
          }
        }}
      >
        <div className="md:px-24 lg:px-48 xl:px-64">
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
                    <Link to={ROUTES.runs.list(tenantId)}>
                      <Button variant="ghost">Skip Tutorial</Button>
                    </Link>
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
      <div className="p-4 bg-background flex flex-row justify-between">
        <div></div>
        <div className="flex flex-row gap-2">
          <Button
            variant="outline"
            disabled={currentStepIndex === 0}
            onClick={() => {
              setActiveStep(stepKeys[currentStepIndex - 1]);
            }}
          >
            Prev
          </Button>
          <Button
            disabled={currentStepIndex === stepKeys.length - 1}
            onClick={() => {
              setActiveStep(stepKeys[currentStepIndex + 1]);
            }}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
