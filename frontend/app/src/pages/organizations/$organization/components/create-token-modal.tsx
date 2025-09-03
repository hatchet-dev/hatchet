import { Button } from '@/components/v1/ui/button';
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
import { useState, useEffect } from 'react';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { KeyIcon, ClipboardIcon, CheckIcon } from '@heroicons/react/24/outline';
import { ManagementTokenDuration } from '@/lib/api/generated/cloud/data-contracts';

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
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

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

  const createTokenMutation = useMutation({
    mutationFn: async (data: {
      name: string;
      duration: ManagementTokenDuration;
    }) => {
      const result = await cloudApi.managementTokenCreate(organizationId, {
        name: data.name,
        duration: data.duration,
      });
      return result.data;
    },
    onSuccess: (data) => {
      setCreatedToken(data.token);
      onSuccess();
    },
    onError: handleApiError,
  });

  const copyToClipboard = async () => {
    if (createdToken) {
      try {
        await navigator.clipboard.writeText(createdToken);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      } catch (err) {
        console.error('Failed to copy: ', err);
      }
    }
  };

  const handleClose = () => {
    setCreatedToken(null);
    setCopied(false);
    reset();
    setFieldErrors({});
    onOpenChange(false);
  };

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setCreatedToken(null);
      setCopied(false);
      reset();
      setFieldErrors({});
    }
  }, [open, reset]);

  const nameError = errors.name?.message?.toString() || fieldErrors?.name;
  const durationError =
    errors.duration?.message?.toString() || fieldErrors?.duration;

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
                <Button
                  variant="outline"
                  size="sm"
                  onClick={copyToClipboard}
                  className="px-3"
                >
                  {copied ? (
                    <CheckIcon className="h-4 w-4 text-green-600" />
                  ) : (
                    <ClipboardIcon className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            <div className="flex items-center justify-end pt-4">
              <Button onClick={handleClose}>Done</Button>
            </div>
          </div>
        ) : (
          <form
            onSubmit={handleSubmit((data) => createTokenMutation.mutate(data))}
            className="space-y-4"
          >
            <div className="space-y-2">
              <Label htmlFor="name">Token Name</Label>
              <Input
                {...register('name')}
                id="name"
                placeholder="e.g., CI/CD Pipeline Token"
                disabled={createTokenMutation.isPending}
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
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={createTokenMutation.isPending}>
                {createTokenMutation.isPending ? 'Creating...' : 'Create Token'}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
