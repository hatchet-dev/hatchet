import { useSandboxProviderMutations } from '../hooks/use-sandbox-providers';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { SandboxProvider } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { AxiosError } from 'axios';

const EFFECTS = [
  {
    title: 'Every sandbox task run on this provider fails.',
    detail:
      'Creates, connects, commands and file operations all stop until the provider is enabled again.',
  },
  {
    title: 'Existing sandboxes stay at the vendor.',
    detail: 'Their idle timeout pauses them. Nothing is deleted.',
  },
];

export const DisableProviderDialog = ({
  provider,
  open,
  onOpenChange,
}: {
  provider: SandboxProvider;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) => {
  const { update } = useSandboxProviderMutations();
  const { handleApiError } = useApiError({});

  return (
    <ConfirmDialog
      isOpen={open}
      title={`Disable ${provider.name} provider`}
      description={
        <div className="space-y-4">
          <p>You can enable it again at any time.</p>
          <ul className="space-y-3 border-l-2 border-border pl-4">
            {EFFECTS.map((effect) => (
              <li key={effect.title}>
                <span className="block font-medium">{effect.title}</span>
                <span className="block text-sm text-muted-foreground">
                  {effect.detail}
                </span>
              </li>
            ))}
          </ul>
        </div>
      }
      submitLabel="Disable provider"
      isLoading={update.isPending}
      onCancel={() => onOpenChange(false)}
      onSubmit={() =>
        update.mutate(
          { providerId: provider.metadata.id, data: { enabled: false } },
          {
            onSuccess: () => onOpenChange(false),
            onError: (error) => handleApiError(error as AxiosError),
          },
        )
      }
    />
  );
};
