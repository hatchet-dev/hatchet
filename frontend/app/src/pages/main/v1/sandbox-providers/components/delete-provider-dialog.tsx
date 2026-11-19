import { useSandboxProviderMutations } from '../hooks/use-sandbox-providers';
import { KINDS } from '../lib/kinds';
import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading';
import { SandboxProvider } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { AxiosError } from 'axios';
import { Info } from 'lucide-react';
import { useEffect, useState } from 'react';

const EFFECTS = [
  {
    title: 'Every sandbox on this provider is deleted.',
    detail: 'The SDK will not be able to connect to them anymore.',
  },
  {
    title: 'Every snapshot is gone.',
    detail: 'Forks and restores from these snapshots will fail.',
  },
  {
    title: 'Runs that used these sandboxes cannot be replayed.',
    detail: "Their history stays visible, but they can't be run again.",
  },
];

export const DeleteProviderDialog = ({
  provider,
  open,
  onOpenChange,
  onDeleted,
}: {
  provider: SandboxProvider;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) => {
  const { remove } = useSandboxProviderMutations();
  const { handleApiError } = useApiError({});
  const [checked, setChecked] = useState<boolean[]>(EFFECTS.map(() => false));
  const [typed, setTyped] = useState('');

  useEffect(() => {
    if (open) {
      setChecked(EFFECTS.map(() => false));
      setTyped('');
    }
  }, [open]);

  const confirmed = checked.every(Boolean) && typed.trim() === provider.name;

  const submit = () => {
    if (!confirmed || remove.isPending) {
      return;
    }
    remove.mutate(provider.metadata.id, {
      onSuccess: () => {
        onOpenChange(false);
        onDeleted?.();
      },
      onError: (error) => handleApiError(error as AxiosError),
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="text-xl">
            Delete {provider.name} provider
          </DialogTitle>
          <DialogDescription>This action cannot be undone.</DialogDescription>
        </DialogHeader>

        <div className="space-y-5 py-2">
          <div className="space-y-4">
            {EFFECTS.map((effect, i) => (
              <label
                key={effect.title}
                className="flex cursor-pointer flex-row items-start gap-3"
              >
                <Checkbox
                  className="mt-1"
                  checked={checked[i]}
                  onCheckedChange={(value) =>
                    setChecked((c) => c.map((v, j) => (j === i ? !!value : v)))
                  }
                />
                <span>
                  <span className="block font-medium">{effect.title}</span>
                  <span className="block text-sm text-muted-foreground">
                    {effect.detail}
                  </span>
                </span>
              </label>
            ))}
          </div>

          <div className="flex flex-row items-center gap-3 rounded-lg border border-blue-500/50 bg-blue-500/10 px-4 py-3 text-sm">
            <Info className="size-4 shrink-0 text-blue-500" />
            <span>
              Machines and snapshots in your {KINDS[provider.kind].label}{' '}
              account are not affected.
            </span>
          </div>

          <div className="space-y-2">
            <p>
              Type <span className="font-mono">{provider.name}</span> to
              confirm.
            </p>
            <Input
              value={typed}
              placeholder={provider.name}
              autoComplete="off"
              data-1p-ignore
              className="h-11"
              onChange={(e) => setTyped(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  submit();
                }
              }}
            />
          </div>
        </div>

        <div className="flex flex-row justify-end gap-3">
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={!confirmed || remove.isPending}
            onClick={submit}
          >
            {remove.isPending && <Spinner />}
            Delete provider
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};
