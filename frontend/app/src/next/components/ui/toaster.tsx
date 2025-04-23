import { useToast } from '@/next/hooks/utils/use-toast';
import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastTitle,
  ToastViewport,
  ToastProvider as RadixToastProvider,
} from '@/next/components/ui/toast';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog';
import { Code } from '@/next/components/ui/code/code';
import { Button } from '@/next/components/ui/button';
import { AlertTriangle } from 'lucide-react';
import { useState } from 'react';

export function Toaster() {
  const { toasts } = useToast();
  const [errorDialogOpen, setErrorDialogOpen] = useState(false);
  const [currentError, setCurrentError] = useState<Error | string | null>(null);

  const handleErrorClick = (error: Error | string) => {
    setCurrentError(error);
    setErrorDialogOpen(true);
  };

  return (
    <>
      <RadixToastProvider>
        {toasts.map(function ({
          id,
          title,
          description,
          action,
          error,
          ...props
        }) {
          return (
            <Toast key={id} {...props}>
              <div className="grid gap-1">
                {title && <ToastTitle>{title}</ToastTitle>}
                {description && (
                  <ToastDescription>{description}</ToastDescription>
                )}
                {!description && error instanceof Error && (
                  <Button
                    variant="link"
                    size="sm"
                    className="justify-start p-0"
                    onClick={() => handleErrorClick(error)}
                  >
                    {error.message}
                  </Button>
                )}
              </div>
              {error && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 px-2"
                  onClick={() => handleErrorClick(error)}
                >
                  <AlertTriangle className="h-4 w-4" />
                </Button>
              )}
              {action}
              <ToastClose />
            </Toast>
          );
        })}
        <ToastViewport />
      </RadixToastProvider>

      <Dialog open={errorDialogOpen} onOpenChange={setErrorDialogOpen}>
        <DialogContent className="max-w-[800px] w-full">
          <DialogHeader className="pb-2">
            <DialogTitle className="text-lg">Error Details</DialogTitle>
          </DialogHeader>
          {currentError && (
            <div className="max-h-[70vh] overflow-auto">
              <Code
                language="json"
                value={
                  typeof currentError === 'string'
                    ? currentError
                    : JSON.stringify(currentError, null, 2)
                }
                showLineNumbers
                className="text-sm"
              />
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
