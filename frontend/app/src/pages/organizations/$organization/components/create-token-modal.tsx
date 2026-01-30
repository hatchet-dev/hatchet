import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useOrganizations } from '@/hooks/use-organizations';
import { ManagementTokenDuration } from '@/lib/api/generated/cloud/data-contracts';
import { KeyIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { zodResolver } from '@hookform/resolvers/zod';
import { useState, useEffect, useCallback } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  name: z.string().min(1, 'Token name is required'),
  duration: z.string(),
});

interface CreateTokenModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  organizationName: string;
  onSuccess: () => void;
}

export function CreateTokenModal({
  open,
  onOpenChange,
  organizationId,
  organizationName,
  onSuccess,
}: CreateTokenModalProps) {
  const DURATION_NEVER = 'never';
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const { handleCreateToken, createTokenLoading } = useOrganizations();

  const {
    register,
    handleSubmit,
    reset,
    control,
    watch,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      duration: ManagementTokenDuration.Value30D,
    },
  });

  const handleClose = useCallback(() => {
    setCreatedToken(null);
    reset();
    onOpenChange(false);
  }, [reset, onOpenChange]);

  const handleTokenCreate = useCallback(
    (data: { name: string; duration?: string }) => {
      const duration =
        data.duration === DURATION_NEVER ? undefined : data.duration;
      handleCreateToken(organizationId, data.name, duration, (tokenData) => {
        setCreatedToken(tokenData.token);
        onSuccess();
      });
    },
    [organizationId, handleCreateToken, onSuccess],
  );

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      handleClose();
    }
  }, [open, handleClose]);

  const nameError = errors.name?.message?.toString();
  const durationError = errors.duration?.message?.toString();
  const selectedDuration = watch('duration');

  const durationOptions = [
    { value: ManagementTokenDuration.Value30D, label: '30 days' },
    { value: ManagementTokenDuration.Value60D, label: '60 days' },
    { value: ManagementTokenDuration.Value90D, label: '90 days' },
    { value: DURATION_NEVER, label: 'Do not expire' },
  ];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <KeyIcon className="h-5 w-5" />
            Create Management Token
          </DialogTitle>
          <DialogDescription>
            Create a new API token for managing {organizationName}
          </DialogDescription>
        </DialogHeader>

        {createdToken ? (
          <div className="space-y-4">
            <div className="rounded-md border border-green-200 bg-green-50 p-4">
              <div className="mb-2 text-sm text-green-800">
                <strong>Token created successfully!</strong>
              </div>
              <p className="text-xs text-green-700">
                Make sure to copy your token now. You won't be able to see it
                again!
              </p>
            </div>

            <div className="space-y-2">
              <Label>Your Management Token</Label>
              <div className="flex items-center gap-2">
                <Input
                  value={createdToken}
                  readOnly
                  className="font-mono text-sm"
                />
                <CopyToClipboard text={createdToken} className="px-3" />
              </div>
            </div>

            <div className="flex items-center justify-end pt-4">
              <Button onClick={handleClose}>Done</Button>
            </div>
          </div>
        ) : (
          <form
            onSubmit={handleSubmit(handleTokenCreate)}
            className="space-y-4"
          >
            <div className="space-y-2">
              <Label htmlFor="name">Token Name</Label>
              <Input
                {...register('name')}
                id="name"
                placeholder="e.g., CI/CD Pipeline Token"
                disabled={createTokenLoading}
              />
              {nameError && (
                <div className="text-sm text-red-500">{nameError}</div>
              )}
              <p className="text-sm text-muted-foreground">
                Choose a descriptive name for your token.
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="duration">Duration</Label>
              <Controller
                control={control}
                name="duration"
                render={({ field }) => (
                  <Select onValueChange={field.onChange} value={field.value}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select duration..." />
                    </SelectTrigger>
                    <SelectContent>
                      {durationOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
              {durationError && (
                <div className="text-sm text-red-500">{durationError}</div>
              )}
              <p className="text-sm text-muted-foreground">
                How long the token should remain valid.
              </p>
              {selectedDuration === DURATION_NEVER && (
                <div className="flex items-start gap-2 rounded-md border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800">
                  <ExclamationTriangleIcon className="mt-0.5 h-4 w-4 flex-shrink-0" />
                  <span>
                    Tokens that never expire pose a security risk. Consider
                    using a shorter duration and rotating tokens regularly.
                  </span>
                </div>
              )}
            </div>

            <div className="flex items-center justify-end gap-3 pt-4">
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={createTokenLoading}>
                {createTokenLoading ? 'Creating...' : 'Create Token'}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
