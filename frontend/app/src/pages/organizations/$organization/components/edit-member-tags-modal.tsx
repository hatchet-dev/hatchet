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

interface EditMemberTagsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  memberId: string;
  memberEmail: string;
  onSuccess: () => void;
}

export function EditMemberTagsModal({
  open,
  onOpenChange,
  organizationId,
  memberId,
  memberEmail,
  onSuccess,
}: EditMemberTagsModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});
  const [rawInput, setRawInput] = useState('');

  const tagsQuery = useQuery({
    ...orgApi.memberTagsGetQuery(organizationId, memberId),
    enabled: open,
  });

  useEffect(() => {
    if (tagsQuery.data) {
      setRawInput(tagsQuery.data.tags.join(', '));
    }
  }, [tagsQuery.data]);

  const setTagsMutation = useMutation({
    ...orgApi.memberTagsSetMutation(organizationId, memberId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['organization-member:list:tags', organizationId, memberId],
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
          <DialogTitle>Edit Member Tags</DialogTitle>
          <DialogDescription>
            Tags for <strong>{memberEmail}</strong>. This member will have
            access to tenants whose tags are a subset of these tags.
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
              Comma-separated list of tags. Leave empty to restrict this member
              to tenants with the "*" tag only.
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
