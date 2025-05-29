import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from 'react';
import docMetadata from '@/next/lib/docs';
import {
  RunDetailSheet,
  RunDetailSheetSerializableProps,
} from '../pages/authenticated/dashboard/runs/detail-sheet/run-detail-sheet';

export const pages = docMetadata;

export type DocRef = {
  title: string;
  href: string;
};

type SidePanelContent = {
  component: React.ReactNode;
  title: React.ReactNode;
  actions?: React.ReactNode;
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
      type: 'run-details';
      content: RunDetailSheetSerializableProps;
    };

export const baseDocsUrl = 'https://docs.hatchet.run';

export function useSidePanelData(): SidePanelData {
  const [isOpen, setIsOpen] = useState(false);
  const [props, setProps] = useState<UseSidePanelProps | null>(null);

  const content = useMemo((): SidePanelContent | null => {
    if (!props) {
      return null;
    }

    const panelType = props.type;

    switch (panelType) {
      case 'run-details':
        return {
          component: <RunDetailSheet {...props.content} />,
          title: 'Run Detail',
        };
      case 'docs':
        return {
          component: (
            <iframe
              src={props.content.href}
              className="absolute inset-0 w-full h-full rounded-md border"
              title={`Documentation: ${props.content.title}`}
              loading="lazy"
            />
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
      console.log('Calling open with props:', props);
      setProps(props);
      setIsOpen(true);
    },
    [setIsOpen],
  );

  console.log({ props, isOpen });

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
