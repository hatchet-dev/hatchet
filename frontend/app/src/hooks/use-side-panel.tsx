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
} from '@/pages/main/workflow-runs/$run/v2components/step-run-detail/step-run-detail';
import { DocPage } from '@/components/docs/docs-button';
import { V1Event, V1Filter } from '@/lib/api';
import { FilterDetailView } from '@/pages/main/filters/components/filter-detail-view';
import { ExpandedEventContent } from '@/pages/main/events';
import { useTheme } from '@/components/theme-provider';

type SidePanelContent =
  | {
      isDocs: false;
      component: React.ReactNode;
      actions?: React.ReactNode;
    }
  | {
      isDocs: true;
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
      queryParams?: Record<string, string>;
      // fixme: make this type safe based on the hashes available in the doc
      scrollTo?: string;
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
    }
  | {
      type: 'filter-detail';
      content: {
        filter: V1Filter;
      };
    };

function useSidePanelData(): SidePanelData {
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
        };
      case 'event-details':
        return {
          isDocs: false,
          component: <ExpandedEventContent event={props.content.event} />,
        };
      case 'filter-detail':
        return {
          isDocs: false,
          component: (
            <FilterDetailView filterId={props.content.filter.metadata.id} />
          ),
        };
      case 'docs':
        const query = props.queryParams ?? {};
        query.theme = theme;

        const queryString = new URLSearchParams(query).toString();
        const url =
          `${props.content.href}?${queryString}` +
          (props.scrollTo ? `#${props.scrollTo}` : '');

        return {
          isDocs: true,
          component: (
            <div className="p-4 size-full">
              <iframe
                src={url}
                className="inset-0 w-full rounded-md border border-slate-800 size-full"
                loading="lazy"
              />
            </div>
          ),
        };
      default:
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
