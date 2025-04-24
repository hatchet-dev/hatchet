import {
  useMemo,
  useState,
  useRef,
  useEffect,
  createContext,
  useContext,
} from 'react';
import {
  HighlightStateMap,
  SupportedLanguage,
  LessonPlan,
  PackageManager,
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
  const [activeStep, setActiveStep] = useState<S>(
    Object.keys(lesson.steps)[0] as S,
  );
  const [extra, setExtra] = useState(lesson.extraDefaults);
  const [highlights, setHighlightState] = useState<HighlightStateMap<S>>({});

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

  useEffect(() => {
    const activeBlock = codeBlocksRef.current[activeStep];
    const activeCard = stepCardsRef.current[activeStep];

    // Add scroll margin to both elements
    if (activeBlock) {
      activeBlock.style.scrollMarginTop = '2rem';
      activeBlock.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
    if (activeCard) {
      activeCard.style.scrollMarginTop = '2rem';
      activeCard.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [activeStep]);

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

  const stepKeys = Object.keys(lesson.steps) as S[];
  const currentStepIndex = stepKeys.indexOf(activeStep);

  const setExtraWithLanguage = (e: Partial<E>) =>
    setExtra((prev) => ({
      ...prev,
      [language]: { ...prev[language], ...e },
    }));

  const setHighlights = (highlightState?: HighlightStateMap<S>) =>
    setHighlightState(highlightState || {});

  const value = {
    language,
    setLanguage,
    activeStep,
    setActiveStep,
    extra: extra[language],
    setExtra: setExtraWithLanguage,
    highlights,
    setHighlights,
    commands,
    stepKeys,
    currentStepIndex,
    codeBlocksRef,
    stepCardsRef,
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
