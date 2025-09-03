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
import { useState, useEffect } from 'react';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';

interface AddTenantModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  organizationName: string;
  onSuccess: () => void;
}

export function AddTenantModal({
  open,
  onOpenChange,
  organizationId,
  organizationName,
  onSuccess,
}: AddTenantModalProps) {
  const [tenantName, setTenantName] = useState('');
  const [tenantSlug, setTenantSlug] = useState('');
  const [isSlugManuallyEdited, setIsSlugManuallyEdited] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const generateSlug = (name: string): string => {
    return (
      name
        .toLowerCase()
        .replace(/[^a-z0-9-]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '') +
      '-' +
      Math.random().toString(36).substring(0, 5)
    );
  };

  // Auto-generate slug when name changes (only if user hasn't manually edited slug)
  useEffect(() => {
    if (tenantName && !isSlugManuallyEdited) {
      const newSlug = generateSlug(tenantName);
      setTenantSlug(newSlug);
    }
  }, [tenantName, isSlugManuallyEdited]);

  const createTenantMutation = useMutation({
    mutationFn: async (data: { name: string; slug: string }) => {
      const result = await cloudApi.organizationCreateTenant(organizationId, {
        name: data.name,
        slug: data.slug,
      });
      return result.data;
    },
    onSuccess: () => {
      resetForm();
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const resetForm = () => {
    setTenantName('');
    setTenantSlug('');
    setIsSlugManuallyEdited(false);
    setFieldErrors({});
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (tenantName.trim() && tenantSlug.trim()) {
      createTenantMutation.mutate({
        name: tenantName.trim(),
        slug: tenantSlug.trim(),
      });
    }
  };

  const handleSlugChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setTenantSlug(e.target.value);
    setIsSlugManuallyEdited(true);
  };

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      resetForm();
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Add New Tenant</DialogTitle>
          <DialogDescription>
            Add a new tenant to {organizationName}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="tenantName">Tenant Name</Label>
            <Input
              id="tenantName"
              type="text"
              placeholder="Enter tenant name"
              value={tenantName}
              onChange={(e) => setTenantName(e.target.value)}
              required
            />
            {fieldErrors.name && (
              <div className="text-sm text-red-500">{fieldErrors.name}</div>
            )}
            <p className="text-sm text-muted-foreground">
              Choose a descriptive name for your tenant.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="tenantSlug">Tenant Slug</Label>
            <Input
              id="tenantSlug"
              type="text"
              placeholder="Auto-generated from name"
              value={tenantSlug}
              onChange={handleSlugChange}
              required
            />
            {fieldErrors.slug && (
              <div className="text-sm text-red-500">{fieldErrors.slug}</div>
            )}
            <p className="text-sm text-muted-foreground">
              A unique identifier for your tenant (auto-generated, but you can
              edit it).
            </p>
          </div>

          <div className="flex items-center justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={
                !tenantName.trim() ||
                !tenantSlug.trim() ||
                createTenantMutation.isPending
              }
            >
              {createTenantMutation.isPending ? 'Creating...' : 'Create Tenant'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
