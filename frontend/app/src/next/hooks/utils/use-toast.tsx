import * as React from 'react';
import type { ToastActionElement, ToastProps } from '@/components/ui/toast';

const TOAST_LIMIT = 1;
const TOAST_REMOVE_DELAY = 1000000;

type ToasterToast = ToastProps & {
  id: string;
  title?: React.ReactNode;
  description?: React.ReactNode;
  action?: ToastActionElement;
  error?: Error | string;
};

type Toast = Omit<ToasterToast, 'id'>;

interface ToastContextType {
  toasts: ToasterToast[];
  toast: (props: Toast) => { id: string; dismiss: () => void };
  dismiss: (toastId?: string) => void;
}

const ToastContext = React.createContext<ToastContextType | undefined>(
  undefined,
);

let count = 0;

function genId() {
  count = (count + 1) % Number.MAX_SAFE_INTEGER;
  return count.toString();
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = React.useState<ToasterToast[]>([]);

  const toast = React.useCallback((props: Toast) => {
    console.log('toast', props);
    const id = genId();

    setToasts((prev) => {
      const newToast = {
        ...props,
        id,
        open: true,
        onOpenChange: (open: boolean) => {
          if (!open) {
            dismiss(id);
          }
        },
      };
      return [newToast, ...prev].slice(0, TOAST_LIMIT);
    });

    return {
      id,
      dismiss: () => dismiss(id),
    };
  }, []);

  const dismiss = React.useCallback((toastId?: string) => {
    setToasts((prev) => {
      if (toastId === undefined) {
        return [];
      }
      return prev.map((t) => (t.id === toastId ? { ...t, open: false } : t));
    });

    if (toastId) {
      setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== toastId));
      }, TOAST_REMOVE_DELAY);
    }
  }, []);

  const value = React.useMemo(
    () => ({
      toasts,
      toast,
      dismiss,
    }),
    [toasts, toast, dismiss],
  );

  return (
    <ToastContext.Provider value={value}>{children}</ToastContext.Provider>
  );
}

export function useToast() {
  const context = React.useContext(ToastContext);
  if (context === undefined) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}
