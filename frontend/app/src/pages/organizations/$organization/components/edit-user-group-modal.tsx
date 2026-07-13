import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { InlineError } from '@/components/v1/ui/inline-error';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import {
  OrganizationMember,
  TenantMemberRoleType,
  UserGroup,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { TrashIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useState } from 'react';

interface EditUserGroupModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  group: UserGroup;
  allOrgMembers: OrganizationMember[];
  allTenantTags: string[];
}

export function EditUserGroupModal({
  open,
  onOpenChange,
  organizationId,
  group,
  allOrgMembers,
  allTenantTags,
}: EditUserGroupModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const [formErrors, setFormErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setFormErrors });

  const groupId = group.metadata.id;

  const [name, setName] = useState(group.name);
  const [role, setRole] = useState<TenantMemberRoleType>(group.role);
  const [tags, setTags] = useState<string[]>(group.tags);

  useEffect(() => {
    if (open) {
      setName(group.name);
      setRole(group.role);
      setTags(group.tags);
    }
  }, [open, group]);

  const membersQuery = useQuery({
    ...orgApi.userGroupMembersListQuery(organizationId, groupId),
    enabled: open,
  });

  const invalidateGroup = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['organization:user-groups:list', organizationId],
    });
    queryClient.invalidateQueries({
      queryKey: [
        'organization:user-group:members:list',
        organizationId,
        groupId,
      ],
    });
  }, [queryClient, organizationId, groupId]);

  const updateMutation = useMutation({
    ...orgApi.userGroupUpdateMutation(organizationId, groupId),
    onSuccess: invalidateGroup,
    onError: handleApiError,
  });

  const tagsMutation = useMutation({
    ...orgApi.userGroupTagsSetMutation(organizationId, groupId),
    onSuccess: invalidateGroup,
    onError: handleApiError,
  });

  const addMemberMutation = useMutation({
    ...orgApi.userGroupMemberAddMutation(organizationId, groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [
          'organization:user-group:members:list',
          organizationId,
          groupId,
        ],
      });
    },
    onError: handleApiError,
  });

  const removeMemberMutation = useMutation({
    ...orgApi.userGroupMemberRemoveMutation(organizationId, groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [
          'organization:user-group:members:list',
          organizationId,
          groupId,
        ],
      });
    },
    onError: handleApiError,
  });

  const currentMembers = membersQuery.data?.rows ?? [];
  const currentMemberIds = new Set(currentMembers.map((m) => m.metadata.id));
  const availableToAdd = allOrgMembers.filter(
    (m) => !currentMemberIds.has(m.metadata.id),
  );

  const availableTagsToAdd = allTenantTags.filter((t) => !tags.includes(t));

  const addTag = (tag: string) => setTags((prev) => [...prev, tag]);
  const removeTag = (tag: string) =>
    setTags((prev) => prev.filter((t) => t !== tag));

  const updateGroup = useCallback(
    (data: { name?: string; role: TenantMemberRoleType }) =>
      updateMutation.mutateAsync(data),
    [updateMutation],
  );

  const saveGroupTags = useCallback(
    (tagsToSave: string[]) => tagsMutation.mutateAsync(tagsToSave),
    [tagsMutation],
  );

  const addMember = useCallback(
    (memberId: string) => addMemberMutation.mutate(memberId),
    [addMemberMutation],
  );

  const removeMember = useCallback(
    (memberId: string) => removeMemberMutation.mutate(memberId),
    [removeMemberMutation],
  );

  const isPending = updateMutation.isPending || tagsMutation.isPending;
  const isMemberMutating =
    addMemberMutation.isPending || removeMemberMutation.isPending;

  const handleSave = useCallback(async () => {
    try {
      await Promise.all([
        updateGroup({ name: name.trim() || undefined, role }),
        saveGroupTags(tags),
      ]);
      onOpenChange(false);
    } catch {
      // errors already shown via onError toast handlers
    }
  }, [updateGroup, saveGroupTags, name, role, tags, onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit User Group</DialogTitle>
        </DialogHeader>
        <div className="space-y-6">
          <InlineError errors={formErrors} />
          {/* Name & Role */}
          <div className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="edit-group-name">Name</Label>
              <Input
                id="edit-group-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isPending}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-group-role">Tenant role</Label>
              <Select
                value={role}
                onValueChange={(v) => setRole(v as TenantMemberRoleType)}
                disabled={isPending}
              >
                <SelectTrigger id="edit-group-role">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={TenantMemberRoleType.MEMBER}>
                    MEMBER
                  </SelectItem>
                  <SelectItem value={TenantMemberRoleType.ADMIN}>
                    ADMIN
                  </SelectItem>
                  <SelectItem value={TenantMemberRoleType.OWNER}>
                    OWNER
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Tags */}
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
            {availableTagsToAdd.length > 0 ? (
              <Select onValueChange={addTag} disabled={isPending} value="">
                <SelectTrigger>
                  <SelectValue placeholder="Add a tag…" />
                </SelectTrigger>
                <SelectContent>
                  {availableTagsToAdd.map((tag) => (
                    <SelectItem key={tag} value={tag}>
                      {tag}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <p className="text-xs text-muted-foreground">
                {allTenantTags.length === 0
                  ? 'No tags exist on any tenant yet.'
                  : 'All available tags have been added.'}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              Members in this group get access to tenants whose tags are a
              subset of these tags.
            </p>
          </div>

          {/* Members */}
          <div className="space-y-2">
            <Label>Members</Label>
            {membersQuery.isLoading ? (
              <p className="text-sm text-muted-foreground">Loading...</p>
            ) : (
              <div className="space-y-1 rounded-md border p-2">
                {currentMembers.length === 0 && (
                  <p className="py-2 text-center text-sm text-muted-foreground">
                    No members yet.
                  </p>
                )}
                {currentMembers.map((m) => (
                  <div
                    key={m.metadata.id}
                    className="flex items-center justify-between rounded px-2 py-1 hover:bg-muted/30"
                  >
                    <span className="font-mono text-sm">{m.email}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0"
                      onClick={() => removeMember(m.metadata.id)}
                      disabled={isMemberMutating}
                    >
                      <TrashIcon className="size-3.5 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {availableToAdd.length > 0 && (
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground">Add member:</p>
                <Select
                  onValueChange={addMember}
                  disabled={isMemberMutating}
                  value=""
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select a member to add..." />
                  </SelectTrigger>
                  <SelectContent>
                    {availableToAdd.map((m) => (
                      <SelectItem key={m.metadata.id} value={m.metadata.id}>
                        {m.email}{' '}
                        <Badge variant="outline" className="ml-1 text-xs">
                          {m.role}
                        </Badge>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>

          <div className="flex justify-end">
            <Button onClick={handleSave} disabled={isPending}>
              {isPending ? 'Saving...' : 'Save'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
