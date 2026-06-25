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
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { KeyboardEvent, useCallback, useEffect, useState } from 'react';

interface EditTenantTagsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  tenantId: string;
  tenantName: string;
  onSuccess: () => void;
  allTenantTags?: string[];
}

export function EditTenantTagsModal({
  open,
  onOpenChange,
  organizationId,
  tenantId,
  tenantName,
  onSuccess,
  allTenantTags = [],
}: EditTenantTagsModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});
  const [tags, setTags] = useState<string[]>([]);
  const [inputValue, setInputValue] = useState('');

  const tagsQuery = useQuery({
    ...orgApi.tenantTagsGetQuery(organizationId, tenantId),
    enabled: open,
  });

  useEffect(() => {
    if (tagsQuery.data) {
      setTags(tagsQuery.data);
    }
  }, [tagsQuery.data]);

  useEffect(() => {
    if (!open) {
      setInputValue('');
    }
  }, [open]);

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

  const addTag = useCallback(
    (value: string) => {
      const trimmed = value.trim();
      if (trimmed && !tags.includes(trimmed)) {
        setTags((prev) => [...prev, trimmed]);
      }
      setInputValue('');
    },
    [tags],
  );

  const removeTag = (tag: string) =>
    setTags((prev) => prev.filter((t) => t !== tag));

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag(inputValue);
    }
  };

  const saveTenantTags = useCallback(
    (tagsToSave: string[]) => setTagsMutation.mutate(tagsToSave),
    [setTagsMutation],
  );

  const isPending = setTagsMutation.isPending;

  const availableTagsToAdd = allTenantTags.filter((t) => !tags.includes(t));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Tenant Tags</DialogTitle>
          <DialogDescription>
            Tags for <strong>{tenantName}</strong>. User groups with these tags
            will have access to this tenant.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Tags</Label>

            {tags.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center gap-1.5 rounded-md border bg-secondary px-3 py-1 text-sm text-secondary-foreground"
                  >
                    {tag}
                    <button
                      type="button"
                      onClick={() => removeTag(tag)}
                      disabled={isPending}
                      className="ml-0.5 rounded hover:text-destructive disabled:opacity-50"
                      aria-label={`Remove ${tag}`}
                    >
                      <XMarkIcon className="size-3.5" />
                    </button>
                  </span>
                ))}
              </div>
            )}

            {/* Select from existing tags */}
            {availableTagsToAdd.length > 0 && (
              <Select onValueChange={addTag} disabled={isPending} value="">
                <SelectTrigger>
                  <SelectValue placeholder="Add an existing tag…" />
                </SelectTrigger>
                <SelectContent>
                  {availableTagsToAdd.map((tag) => (
                    <SelectItem key={tag} value={tag}>
                      {tag}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}

            {/* Add a new tag */}
            <div className="flex gap-2">
              <Input
                id="tag-input"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="New tag..."
                disabled={isPending}
                autoComplete="off"
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => addTag(inputValue)}
                disabled={isPending || !inputValue.trim()}
              >
                Add
              </Button>
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button onClick={() => saveTenantTags(tags)} disabled={isPending}>
              {isPending ? 'Saving...' : 'Save Tags'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
