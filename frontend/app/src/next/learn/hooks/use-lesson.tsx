import {
  useMemo,
  useState,
  useRef,
  useEffect,
  createContext,
  useContext,
  useCallback,
} from 'react';
import {
  HighlightStateMap,
  SupportedLanguage,
  LessonPlan,
  PackageManager,
  HighlightState,
} from '@/next/learn/components';

interface ExtraWithPackageManager {
  packageManager: PackageManager;
}

export interface LessonState<
  S extends string,
  E extends ExtraWithPackageManager,
  C,
> {
  language: SupportedLanguage;
  setLanguage: (language: SupportedLanguage) => void;
  activeStep: S;
  setActiveStep: (step: S) => void;
  extra: Partial<E>;
  setExtra: (e: Partial<E>) => void;
  highlights: HighlightStateMap<S>;
  setHighlights: (state?: HighlightStateMap<S>) => void;
  commands: C;
  stepKeys: S[];
  currentStepIndex: number;
  codeBlocksRef: React.MutableRefObject<Record<S, HTMLDivElement | null>>;
  stepCardsRef: React.MutableRefObject<Record<S, HTMLDivElement | null>>;
  codeKeyFrames: Record<string, Partial<Record<S, HighlightState>>>;
}

export interface LessonProviderProps<
  S extends string,
  E extends ExtraWithPackageManager,
  C,
> {
  children: React.ReactNode;
  lesson: LessonPlan<S, E, C>;
}

const LessonContext = createContext<LessonState<any, any, any> | null>(null);

export function LessonProvider<
  S extends string,
  E extends ExtraWithPackageManager,
  C,
>({ children, lesson }: LessonProviderProps<S, E, C>) {
  const [language, setLanguage] = useState(lesson.defaultLanguage);
  const [activeStep, setActiveStep] = useState<S>();
  const [extra, setExtra] = useState(lesson.extraDefaults);
  const [highlights, setHighlights] = useState<HighlightStateMap<S>>({});

  const codeBlocksRef = useRef<Record<S, HTMLDivElement | null>>(
    Object.keys(lesson.steps).reduce(
      (acc, key) => ({
        ...acc,
        [key]: null,
      }),
      {} as Record<S, HTMLDivElement | null>,
    ),
  );

  const stepCardsRef = useRef<Record<S, HTMLDivElement | null>>(
    Object.keys(lesson.steps).reduce(
      (acc, key) => ({
        ...acc,
        [key]: null,
      }),
      {} as Record<S, HTMLDivElement | null>,
    ),
  );

  const stepKeys = Object.keys(lesson.steps) as S[];
  const currentStepIndex = activeStep ? stepKeys.indexOf(activeStep) : -1;

  useEffect(() => {
    const activeBlock = activeStep ? stepCardsRef.current[activeStep] : null;

    // Add scroll margin to both elements
    if (!activeBlock) {
      return;
    }

    activeBlock.style.scrollMarginTop = '2rem';
    activeBlock.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }, [activeStep]);

  const onSetActiveStep = (step: S) => {
    setActiveStep(step);
  };

  const commands = useMemo(() => {
    const languageExtra = extra[language];
    const defaultExtra = lesson.extraDefaults[language];

    if (!languageExtra || !defaultExtra) {
      throw new Error(`Invalid language: ${language}`);
    }

    const packageManager =
      languageExtra.packageManager || defaultExtra.packageManager;
    if (!packageManager) {
      throw new Error(`Invalid package manager for language: ${language}`);
    }

    return lesson.commands[packageManager];
  }, [language, extra, lesson]);

  const codeKeyFrames = useMemo(() => {
    return lesson.codeKeyFrames[language];
  }, [language, lesson]);

  const setExtraWithLanguage = (e: Partial<E>) =>
    setExtra((prev) => ({
      ...prev,
      [language]: { ...prev[language], ...e },
    }));

  const onSetHighlights = useCallback(
    (highlightState?: HighlightStateMap<S>) =>
      setHighlights(highlightState || {}),
    [setHighlights],
  );

  const value = {
    language,
    setLanguage,
    activeStep,
    setActiveStep: onSetActiveStep,
    extra: extra[language],
    setExtra: setExtraWithLanguage,
    highlights,
    setHighlights: onSetHighlights,
    commands,
    stepKeys,
    currentStepIndex,
    codeBlocksRef,
    stepCardsRef,
    codeKeyFrames,
  };

  return (
    <LessonContext.Provider value={value}>{children}</LessonContext.Provider>
  );
}

export function useLesson<
  S extends string,
  E extends ExtraWithPackageManager,
  C,
>(): LessonState<S, E, C> {
  const context = useContext(LessonContext);
  if (!context) {
    throw new Error('useLesson must be used within a LessonProvider');
  }
  return context;
}
