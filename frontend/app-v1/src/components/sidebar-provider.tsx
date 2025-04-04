import {
  PropsWithChildren,
  createContext,
  useContext,
  useEffect,
  useState,
} from 'react';

type SidebarState = 'open' | 'closed';

type SidebarProviderProps = PropsWithChildren & {
  defaultSidebarOpen?: SidebarState;
};

type SidebarProviderState = {
  sidebarOpen: SidebarState;
  setSidebarOpen: (open: SidebarState) => void;
  toggleSidebarOpen: () => void;
};

const initialState: SidebarProviderState = {
  sidebarOpen: 'closed',
  setSidebarOpen: () => null,
  toggleSidebarOpen: () => null,
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

  const [isWide, setIsWide] = useState(false);

  useEffect(() => {
    const handleResize = () => {
      setIsWide(window.innerWidth >= 768);
    };

    handleResize();

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useEffect(() => {
    if (isWide) {
      setSidebarOpen((prev) => (prev ? 'open' : 'closed'));
    } else {
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
