import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { V1Event, V1Filter, ScheduledWorkflows } from '@/lib/api';
import { ExpandedEventContent } from '@/pages/main/v1/events';
import { FilterDetailView } from '@/pages/main/v1/filters/components/filter-detail-view';
import { ExpandedScheduledRunContent } from '@/pages/main/v1/scheduled-runs/components/expanded-scheduled-run-content';
import {
  SpanDetail,
  GroupDetail,
} from '@/pages/main/v1/workflow-runs-v1/$run/v2components/step-run-detail/observability/span-detail';
import type { SpanGroupInfo } from '@/pages/main/v1/workflow-runs-v1/$run/v2components/step-run-detail/observability/timeline/trace-timeline-utils';
import {
  TaskRunDetail,
  TabOption,
} from '@/pages/main/v1/workflow-runs-v1/$run/v2components/step-run-detail/step-run-detail';
import { RunDetailSearchLocalProvider } from '@/pages/main/v1/workflow-runs-v1/hooks/use-run-detail-search';
import { useLocation } from '@tanstack/react-router';
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';

type SidePanelContent = {
  component: React.ReactNode;
  actions?: React.ReactNode;
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
    }
  | {
      type: 'scheduled-run-details';
      content: {
        scheduledRun: ScheduledWorkflows;
      };
    }
  | {
      type: 'span-details';
      content: {
        span: OtelSpanTree;
        onSpanSelect?: (span: OtelSpanTree) => void;
        onClose?: () => void;
      };
    }
  | {
      type: 'group-details';
      content: {
        group: SpanGroupInfo;
        onClose?: () => void;
      };
    };

function SidePanelTaskRunDetail(props: {
  taskRunId: string;
  defaultOpenTab?: TabOption;
  showViewTaskRunButton?: boolean;
}) {
  return (
    <RunDetailSearchLocalProvider>
      <TaskRunDetail {...props} />
    </RunDetailSearchLocalProvider>
  );
}

function SidePanelSpanDetail(
  props: Omit<React.ComponentProps<typeof SpanDetail>, 'onOpenTaskRun'>,
) {
  const { open } = useSidePanel();
  return (
    <SpanDetail
      {...props}
      onOpenTaskRun={(taskRunId) =>
        open({
          type: 'task-run-details',
          content: { taskRunId, showViewTaskRunButton: true },
        })
      }
    />
  );
}

function useSidePanelData(): SidePanelData {
  const [isOpen, setIsOpen] = useState(false);
  const [history, setHistory] = useState<UseSidePanelProps[]>([]);
  const [currentIndex, setCurrentIndex] = useState(-1);
  const location = useLocation();

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
          component: <SidePanelTaskRunDetail {...props.content} />,
        };
      case 'event-details':
        return {
          component: <ExpandedEventContent event={props.content.event} />,
        };
      case 'filter-detail':
        return {
          component: (
            <FilterDetailView filterId={props.content.filter.metadata.id} />
          ),
        };
      case 'scheduled-run-details':
        return {
          component: (
            <ExpandedScheduledRunContent
              scheduledRun={props.content.scheduledRun}
            />
          ),
        };
      case 'span-details':
        return {
          component: <SidePanelSpanDetail {...props.content} />,
        };
      case 'group-details':
        return {
          component: <GroupDetail {...props.content} />,
        };
      default:
        const exhaustiveCheck: never = panelType;
        throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
    }
  }, [props]);

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
