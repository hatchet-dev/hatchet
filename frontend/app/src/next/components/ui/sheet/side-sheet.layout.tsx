import { Cross2Icon } from '@radix-ui/react-icons';
import { Button } from '../button';
import { useSidePanel } from '@/next/hooks/use-side-panel';
import { useLocation } from 'react-router-dom';
import { useEffect, useRef } from 'react';

function useNavigationListener(callback: () => void) {
  const location = useLocation();
  const prevPathname = useRef(location.pathname);

  useEffect(() => {
    if (location.pathname !== prevPathname.current) {
      callback();
      prevPathname.current = location.pathname;
    }
  }, [location.pathname, callback]);
}

export function SidePanel() {
  const { content: maybeContent, isOpen, close } = useSidePanel();

  useNavigationListener(() => {
    if (isOpen) {
      close();
    }
  });

  if (!maybeContent) {
    return null;
  }

  return (
    isOpen && (
      <div className="flex flex-col h-screen">
        <div
          className={
            'flex flex-row w-full justify-between items-center border-b bg-background h-16 px-4 md:px-12'
          }
        >
          <h2 className="text-lg font-semibold truncate pr-2">
            {maybeContent.title}
          </h2>
          <div className="flex items-center gap-2">
            {!maybeContent.isDocs && maybeContent.actions}
            <Button
              variant="ghost"
              onClick={close}
              className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
            >
              <Cross2Icon className="h-4 w-4" />
              <span className="sr-only">Close</span>
            </Button>
          </div>
        </div>

        {maybeContent.component}
      </div>
    )
  );
}
