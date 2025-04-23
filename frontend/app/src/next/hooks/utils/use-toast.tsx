import * as React from 'react';
import type { ToastActionElement, ToastProps } from '@/components/ui/toast';

const TOAST_LIMIT = 3;
const TOAST_REMOVE_DELAY = 1000;
const TOAST_DURATION = 5000;

type ToasterToast = ToastProps & {
  id: string;
  title?: React.ReactNode;
  description?: React.ReactNode;
  action?: ToastActionElement;
  error?: Error | string | unknown;
  persist?: boolean;
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
      // Add new toast to the end of the array and limit to TOAST_LIMIT
      return [...prev, newToast].slice(-TOAST_LIMIT);
    });

    // Auto-dismiss after TOAST_DURATION if not persistent
    if (!props.persist) {
      setTimeout(() => {
        dismiss(id);
      }, TOAST_DURATION);
    }

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
