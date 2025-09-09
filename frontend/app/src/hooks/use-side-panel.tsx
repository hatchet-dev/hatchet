import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { useLocation } from 'react-router-dom';
import {
  TaskRunDetail,
  TabOption,
} from '@/pages/main/v1/workflow-runs-v1/$run/v2components/step-run-detail/step-run-detail';
import { DocPage } from '@/components/v1/docs/docs-button';
import { V1Event } from '@/lib/api';
import { ExpandedEventContent } from '@/pages/main/v1/events';
import { useTheme } from '@/components/theme-provider';

type SidePanelContent =
  | {
      isDocs: false;
      component: React.ReactNode;
      title: React.ReactNode;
      actions?: React.ReactNode;
    }
  | {
      isDocs: true;
      title: string;
      component: React.ReactNode;
    };

type SidePanelData = {
  content: SidePanelContent | null;
  isOpen: boolean;
  open: (props: UseSidePanelProps) => void;
  close: () => void;
  canGoBack: boolean;
  canGoForward: boolean;
  goBack: () => void;
  goForward: () => void;
};

type UseSidePanelProps =
  | {
      type: 'docs';
      content: DocPage;
    }
  | {
      type: 'task-run-details';
      content: {
        taskRunId: string;
        defaultOpenTab?: TabOption;
        showViewTaskRunButton?: boolean;
      };
    }
  | {
      type: 'event-details';
      content: {
        event: V1Event;
      };
    };

export function useSidePanelData(): SidePanelData {
  const [isOpen, setIsOpen] = useState(false);
  const [history, setHistory] = useState<UseSidePanelProps[]>([]);
  const [currentIndex, setCurrentIndex] = useState(-1);
  const location = useLocation();
  const { theme: rawTheme } = useTheme();
  const theme = ['dark', 'light'].includes(rawTheme) ? rawTheme : 'dark';

  const props =
    currentIndex >= 0 && currentIndex < history.length
      ? history[currentIndex]
      : null;

  useEffect(() => {
    setIsOpen(false);
    setHistory([]);
    setCurrentIndex(-1);
  }, [location.pathname]);

  const content = useMemo((): SidePanelContent | null => {
    if (!props) {
      return null;
    }

    const panelType = props.type;

    switch (panelType) {
      case 'task-run-details':
        return {
          isDocs: false,
          component: <TaskRunDetail {...props.content} />,
          title: 'Run details',
        };
      case 'event-details':
        return {
          isDocs: false,
          component: <ExpandedEventContent event={props.content.event} />,
          title: `Event ${props.content.event.key} details`,
        };
      case 'docs':
        return {
          isDocs: true,
          component: (
            <div className="p-4 size-full">
              <iframe
                src={`${props.content.href}?theme=${theme}`}
                className="inset-0 w-full rounded-md border border-slate-800 size-full"
                title={`Documentation: ${props.content.title}`}
                loading="lazy"
              />
            </div>
          ),
          title: props.content.title,
        };
      default:
        // eslint-disable-next-line no-case-declarations
        const exhaustiveCheck: never = panelType;
        throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
    }
  }, [props, theme]);

  const open = useCallback(
    (newProps: UseSidePanelProps) => {
      setHistory((prev) => {
        const newHistory = prev.slice(0, currentIndex + 1);
        newHistory.push(newProps);
        return newHistory;
      });
      setCurrentIndex((prev) => prev + 1);
      setIsOpen(true);
    },
    [currentIndex],
  );

  const close = useCallback(() => {
    setIsOpen(false);
    setHistory([]);
    setCurrentIndex(-1);
  }, []);

  const goBack = useCallback(() => {
    if (currentIndex > 0) {
      setCurrentIndex((prev) => prev - 1);
    }
  }, [currentIndex]);

  const goForward = useCallback(() => {
    if (currentIndex < history.length - 1) {
      setCurrentIndex((prev) => prev + 1);
    }
  }, [currentIndex, history.length]);

  const canGoBack = currentIndex > 0;
  const canGoForward = currentIndex < history.length - 1;

  return {
    isOpen,
    content,
    open,
    close,
    canGoBack,
    canGoForward,
    goBack,
    goForward,
  };
}

const SidePanelContext = createContext<SidePanelData | null>(null);

export function SidePanelProvider({ children }: { children: React.ReactNode }) {
  const sidePanelState = useSidePanelData();

  return (
    <SidePanelContext.Provider value={sidePanelState}>
      {children}
    </SidePanelContext.Provider>
  );
}

export function useSidePanel(): SidePanelData {
  const context = useContext(SidePanelContext);
  if (!context) {
    throw new Error(
      'useSidePanelContext must be used within a SidePanelProvider',
    );
  }
  return context;
}
