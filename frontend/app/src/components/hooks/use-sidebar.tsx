import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import {
  PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';

type SidebarState = 'open' | 'closed';

const SIDEBAR_WIDTH_LEGACY_KEY = 'v1SidebarWidth';
const SIDEBAR_WIDTH_EXPANDED_KEY = 'v1SidebarWidthExpanded';
const SIDEBAR_COLLAPSED_KEY = 'v1SidebarCollapsed';

// Tailwind's `md` breakpoint (px). Used to decide when the v1 sidebar is a persistent column.
const WIDE_BREAKPOINT_PX = 768;

// Widths (px)
const DEFAULT_EXPANDED_SIDEBAR_WIDTH = 200;
export const MIN_EXPANDED_SIDEBAR_WIDTH = 200;
export const MAX_EXPANDED_SIDEBAR_WIDTH = 520;
export const COLLAPSED_SIDEBAR_WIDTH = 56;

// Behavior
export const COLLAPSE_SNAP_AT = MIN_EXPANDED_SIDEBAR_WIDTH;
export const EXPAND_SNAP_AT = MIN_EXPANDED_SIDEBAR_WIDTH - 100;
export const RESIZE_DRAG_THRESHOLD_PX = 3;

type SidebarProviderProps = PropsWithChildren & {
  defaultSidebarOpen?: SidebarState;
};

type SidebarProviderState = {
  sidebarOpen: SidebarState;
  setSidebarOpen: (open: SidebarState) => void;
  toggleSidebarOpen: () => void;
  isWide: boolean;
  sidebarWidth?: number;
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
  toggleCollapsed: () => void;
  expandedWidth: number;
  setExpandedWidth: (width: number) => void;
};

const initialState: SidebarProviderState = {
  sidebarOpen: 'closed',
  setSidebarOpen: () => null,
  toggleSidebarOpen: () => null,
  isWide: false,
  sidebarWidth: undefined,
  collapsed: false,
  setCollapsed: () => null,
  toggleCollapsed: () => null,
  expandedWidth: DEFAULT_EXPANDED_SIDEBAR_WIDTH,
  setExpandedWidth: () => null,
};

const SidebarProviderContext =
  createContext<SidebarProviderState>(initialState);

export function SidebarProvider({
  children,
  defaultSidebarOpen = 'closed',

  ...props
}: SidebarProviderProps) {
  const [sidebarOpen, setSidebarOpen] = useState<SidebarState>(
    () => defaultSidebarOpen,
  );

  // Initialize from the current viewport so we don't "animate" from mobile->desktop
  // width on first load (the resize effect will keep it updated).
  const [isWide, setIsWide] = useState(() =>
    typeof window !== 'undefined'
      ? window.innerWidth >= WIDE_BREAKPOINT_PX
      : false,
  );

  const defaultExpandedWidth = useMemo(() => {
    if (typeof window === 'undefined') {
      return DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    }

    try {
      // Back-compat: previous implementation stored this under `v1SidebarWidth`.
      const legacy = window.localStorage.getItem(SIDEBAR_WIDTH_LEGACY_KEY);
      return legacy ? JSON.parse(legacy) : DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    } catch {
      return DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    }
  }, []);

  const [expandedWidth, setExpandedWidth] = useLocalStorageState(
    SIDEBAR_WIDTH_EXPANDED_KEY,
    defaultExpandedWidth,
  );

  const [collapsed, setCollapsed] = useLocalStorageState(
    SIDEBAR_COLLAPSED_KEY,
    false,
  );

  const sidebarWidth = useMemo(() => {
    if (!isWide) {
      return undefined;
    }

    return collapsed ? COLLAPSED_SIDEBAR_WIDTH : expandedWidth;
  }, [collapsed, expandedWidth, isWide]);

  const toggleCollapsed = useCallback(() => {
    setCollapsed(!collapsed);
  }, [collapsed, setCollapsed]);

  useEffect(() => {
    const handleResize = () => {
      setIsWide(window.innerWidth >= WIDE_BREAKPOINT_PX);
    };

    handleResize();

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useEffect(() => {
    if (isWide) {
      // Desktop: sidebar participates in the layout grid; keep it open.
      setSidebarOpen('open');
    } else {
      // Mobile: default to closed (overlay sidebar).
      setSidebarOpen('closed');
    }
  }, [isWide]);

  return (
    <SidebarProviderContext.Provider
      {...props}
      value={{
        sidebarOpen,
        setSidebarOpen: (open: SidebarState) => {
          setSidebarOpen(open);
        },
        toggleSidebarOpen: () => {
          setSidebarOpen((state) => (state === 'open' ? 'closed' : 'open'));
        },
        isWide,
        sidebarWidth,
        collapsed,
        setCollapsed,
        toggleCollapsed,
        expandedWidth,
        setExpandedWidth,
      }}
    >
      {children}
    </SidebarProviderContext.Provider>
  );
}

export const useSidebar = () => {
  const context = useContext(SidebarProviderContext);

  if (context === undefined) {
    throw new Error('useSidebar must be used within a SidebarProvider');
  }

  return context;
};
