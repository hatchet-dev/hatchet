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

export type DocRef = {
  title: string;
  href: string;
};

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
};

type UseSidePanelProps =
  | {
      type: 'docs';
      content: DocRef;
    }
  | {
      type: 'task-run-details';
      content: {
        taskRunId: string;
        defaultOpenTab?: TabOption;
        showViewTaskRunButton?: boolean;
      };
    };

export function useSidePanelData(): SidePanelData {
  const [isOpen, setIsOpen] = useState(false);
  const [props, setProps] = useState<UseSidePanelProps | null>(null);
  const location = useLocation();

  useEffect(() => {
    setIsOpen(false);
    setProps(null);
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
          title: 'Run Detail',
        };
      case 'docs':
        return {
          isDocs: true,
          component: (
            <div className="p-4 size-full">
              <iframe
                src={props.content.href}
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
  }, [props]);

  const open = useCallback(
    (props: UseSidePanelProps) => {
      setProps(props);
      setIsOpen(true);
    },
    [setIsOpen],
  );

  const close = useCallback(() => {
    setIsOpen(false);
  }, [setIsOpen]);

  return {
    isOpen,
    content,
    open,
    close,
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
