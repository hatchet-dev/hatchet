import { Button, ReviewedButtonTemp } from '@/components/v1/ui/button';
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
import { useState, useEffect, useCallback } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { useOrganizations } from '@/hooks/use-organizations';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { KeyIcon } from '@heroicons/react/24/outline';
import { ManagementTokenDuration } from '@/lib/api/generated/cloud/data-contracts';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';

const schema = z.object({
  name: z.string().min(1, 'Token name is required'),
  duration: z.nativeEnum(ManagementTokenDuration),
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
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const { handleCreateToken, createTokenLoading } = useOrganizations();

  const {
    register,
    handleSubmit,
    reset,
    control,
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
    (data: { name: string; duration: ManagementTokenDuration }) => {
      handleCreateToken(
        organizationId,
        data.name,
        data.duration,
        (tokenData) => {
          setCreatedToken(tokenData.token);
          onSuccess();
        },
      );
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

  const durationOptions = [
    { value: ManagementTokenDuration.Value30D, label: '30 days' },
    { value: ManagementTokenDuration.Value60D, label: '60 days' },
    { value: ManagementTokenDuration.Value90D, label: '90 days' },
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
            <div className="p-4 bg-green-50 border border-green-200 rounded-md">
              <div className="text-sm text-green-800 mb-2">
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
              <ReviewedButtonTemp onClick={handleClose}>
                Done
              </ReviewedButtonTemp>
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
            </div>

            <div className="flex items-center justify-end gap-3 pt-4">
              <ReviewedButtonTemp
                type="button"
                variant="outline"
                onClick={handleClose}
              >
                Cancel
              </ReviewedButtonTemp>
              <ReviewedButtonTemp type="submit" disabled={createTokenLoading}>
                {createTokenLoading ? 'Creating...' : 'Create Token'}
              </ReviewedButtonTemp>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
