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
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useState } from 'react';

interface EditTenantTagsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  tenantId: string;
  tenantName: string;
  onSuccess: () => void;
}

export function EditTenantTagsModal({
  open,
  onOpenChange,
  organizationId,
  tenantId,
  tenantName,
  onSuccess,
}: EditTenantTagsModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});
  const [rawInput, setRawInput] = useState('');

  const tagsQuery = useQuery({
    ...orgApi.tenantTagsGetQuery(organizationId, tenantId),
    enabled: open,
  });

  useEffect(() => {
    if (tagsQuery.data) {
      setRawInput(tagsQuery.data.tags.join(', '));
    }
  }, [tagsQuery.data]);

  const setTagsMutation = useMutation({
    ...orgApi.tenantTagsSetMutation(organizationId, tenantId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['organization-tenant:list:tags', organizationId, tenantId],
      });
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const handleSave = useCallback(() => {
    const tags = rawInput
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean);
    setTagsMutation.mutate(tags);
  }, [rawInput, setTagsMutation]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Tenant Tags</DialogTitle>
          <DialogDescription>
            Tags for <strong>{tenantName}</strong>. Users whose tags include all
            of these tags will have access to this tenant.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="tags">Tags</Label>
            <Input
              id="tags"
              value={rawInput}
              onChange={(e) => setRawInput(e.target.value)}
              placeholder="e.g. prod, us-east"
              disabled={setTagsMutation.isPending}
            />
            <p className="text-sm text-muted-foreground">
              Comma-separated list of tags. Use "*" to allow all members access.
            </p>
          </div>
          <div className="flex justify-end gap-3">
            <Button
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={setTagsMutation.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={setTagsMutation.isPending}>
              {setTagsMutation.isPending ? 'Saving...' : 'Save Tags'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
