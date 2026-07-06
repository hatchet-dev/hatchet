import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/v1/ui/dialog';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';

export function AuthDisabledBanner() {
  const { meta } = useApiMeta();
  const token =
    meta && 'authDisabledToken' in meta ? meta.authDisabledToken : undefined;

  return (
    <div
      role="alert"
      className="flex items-center justify-center gap-3 border-b border-red-300 bg-red-600 px-4 py-1.5 text-center text-sm font-medium text-white dark:border-red-800"
    >
      <ExclamationTriangleIcon className="h-4 w-4 flex-shrink-0" />
      You are using an auth-disabled instance of Hatchet.
      {token && (
        <Dialog>
          <DialogTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className="h-6 border-white/40 bg-transparent px-2 text-xs text-white hover:bg-white/10 hover:text-white"
            >
              View worker token
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Worker API token</DialogTitle>
              <DialogDescription>
                Workers authenticate over gRPC with this built-in token, scoped
                to the default tenant. It is the same on every auth-disabled
                instance.
              </DialogDescription>
            </DialogHeader>
            <CodeHighlighter language="text" code={token} />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}
