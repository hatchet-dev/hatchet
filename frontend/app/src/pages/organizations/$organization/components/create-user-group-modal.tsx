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
import { TenantMemberRoleType } from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';

interface CreateUserGroupModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  allTenantTags?: string[];
}

export function CreateUserGroupModal({
  open,
  onOpenChange,
  organizationId,
  allTenantTags = [],
}: CreateUserGroupModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});
  const [name, setName] = useState('');
  const [role, setRole] = useState<TenantMemberRoleType>(
    TenantMemberRoleType.MEMBER,
  );
  const [tags, setTags] = useState<string[]>([]);
  const [newTag, setNewTag] = useState('');

  const availableTagsToAdd = useMemo(
    () => allTenantTags.filter((tag) => !tags.includes(tag)),
    [allTenantTags, tags],
  );

  const addTag = (tag: string) => {
    const trimmed = tag.trim();
    if (!trimmed || tags.includes(trimmed)) {
      return;
    }
    setTags((prev) => [...prev, trimmed]);
  };

  const removeTag = (tag: string) =>
    setTags((prev) => prev.filter((t) => t !== tag));

  const createMutation = useMutation({
    mutationKey: ['organization:user-groups:create', organizationId],
    mutationFn: async (data: {
      name: string;
      role: TenantMemberRoleType;
      tags: string[];
    }) => {
      // The create endpoint doesn't accept tags, so set them right after.
      // If setting tags fails, the group still exists — the error surfaces
      // and tags can be added from the edit modal.
      const group = await orgApi
        .userGroupCreateMutation(organizationId)
        .mutationFn({ name: data.name, role: data.role });

      if (data.tags.length > 0 && group?.metadata?.id) {
        await orgApi
          .userGroupTagsSetMutation(organizationId, group.metadata.id)
          .mutationFn(data.tags);
      }

      return group;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['organization:user-groups:list', organizationId],
      });
      setName('');
      setRole(TenantMemberRoleType.MEMBER);
      setTags([]);
      setNewTag('');
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const isPending = createMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Create User Group</DialogTitle>
          <DialogDescription>
            User groups associate tags and a tenant role with a set of org
            members.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="group-name">Name</Label>
            <Input
              id="group-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Engineers"
              disabled={isPending}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="group-role">Tenant role</Label>
            <Select
              value={role}
              onValueChange={(v) => setRole(v as TenantMemberRoleType)}
              disabled={isPending}
            >
              <SelectTrigger id="group-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={TenantMemberRoleType.MEMBER}>
                  Member
                </SelectItem>
                <SelectItem value={TenantMemberRoleType.ADMIN}>
                  Admin
                </SelectItem>
                <SelectItem value={TenantMemberRoleType.OWNER}>
                  Owner
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Tags</Label>
            <p className="text-xs text-muted-foreground">
              Members of this group get access to tenants with matching tags.
            </p>
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-xs"
                  >
                    {tag}
                    <button
                      type="button"
                      onClick={() => removeTag(tag)}
                      disabled={isPending}
                      className="text-muted-foreground hover:text-foreground focus:outline-none"
                    >
                      <XMarkIcon className="size-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
            {availableTagsToAdd.length > 0 && (
              <Select
                value=""
                onValueChange={(tag) => addTag(tag)}
                disabled={isPending}
              >
                <SelectTrigger className="text-muted-foreground">
                  <SelectValue placeholder="Add an existing tag" />
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
            <div className="flex gap-2">
              <Input
                value={newTag}
                onChange={(e) => setNewTag(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addTag(newTag);
                    setNewTag('');
                  }
                }}
                placeholder="Or create a new tag"
                disabled={isPending}
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  addTag(newTag);
                  setNewTag('');
                }}
                disabled={isPending || !newTag.trim()}
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
            <Button
              onClick={() => createMutation.mutate({ name, role, tags })}
              disabled={isPending || !name.trim()}
            >
              {isPending ? 'Creating...' : 'Create Group'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
