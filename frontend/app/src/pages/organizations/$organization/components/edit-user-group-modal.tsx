import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
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
import {
  OrganizationMember,
  TenantMemberRoleType,
  UserGroup,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { TrashIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useState } from 'react';

interface EditUserGroupModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  group: UserGroup;
  allOrgMembers: OrganizationMember[];
}

export function EditUserGroupModal({
  open,
  onOpenChange,
  organizationId,
  group,
  allOrgMembers,
}: EditUserGroupModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});

  const groupId = group.metadata.id;

  const [name, setName] = useState(group.name);
  const [role, setRole] = useState<TenantMemberRoleType>(group.role);
  const [rawTags, setRawTags] = useState(group.tags.join(', '));

  useEffect(() => {
    if (open) {
      setName(group.name);
      setRole(group.role);
      setRawTags(group.tags.join(', '));
    }
  }, [open, group]);

  const membersQuery = useQuery({
    ...orgApi.userGroupMembersListQuery(organizationId, groupId),
    enabled: open,
  });

  const invalidateGroup = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['organization:user-groups:list', organizationId] });
    queryClient.invalidateQueries({ queryKey: ['organization:user-group:members:list', organizationId, groupId] });
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
      queryClient.invalidateQueries({ queryKey: ['organization:user-group:members:list', organizationId, groupId] });
    },
    onError: handleApiError,
  });

  const removeMemberMutation = useMutation({
    ...orgApi.userGroupMemberRemoveMutation(organizationId, groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization:user-group:members:list', organizationId, groupId] });
    },
    onError: handleApiError,
  });

  const currentMembers = membersQuery.data?.rows ?? [];
  const currentMemberIds = new Set(currentMembers.map((m) => m.metadata.id));
  const availableToAdd = allOrgMembers.filter((m) => !currentMemberIds.has(m.metadata.id));

  const handleSaveNameRole = () => {
    updateMutation.mutate({ name: name.trim() || undefined, role });
  };

  const handleSaveTags = () => {
    const tags = rawTags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean);
    tagsMutation.mutate(tags);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit User Group</DialogTitle>
        </DialogHeader>
        <div className="space-y-6">
          {/* Name & Role */}
          <div className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="edit-group-name">Name</Label>
              <Input
                id="edit-group-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={updateMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-group-role">Tenant role</Label>
              <Select
                value={role}
                onValueChange={(v) => setRole(v as TenantMemberRoleType)}
                disabled={updateMutation.isPending}
              >
                <SelectTrigger id="edit-group-role">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={TenantMemberRoleType.MEMBER}>MEMBER</SelectItem>
                  <SelectItem value={TenantMemberRoleType.ADMIN}>ADMIN</SelectItem>
                  <SelectItem value={TenantMemberRoleType.OWNER}>OWNER</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex justify-end">
              <Button
                size="sm"
                onClick={handleSaveNameRole}
                disabled={updateMutation.isPending}
              >
                {updateMutation.isPending ? 'Saving...' : 'Save'}
              </Button>
            </div>
          </div>

          {/* Tags */}
          <div className="space-y-2">
            <Label htmlFor="edit-group-tags">Tags</Label>
            <Input
              id="edit-group-tags"
              value={rawTags}
              onChange={(e) => setRawTags(e.target.value)}
              placeholder="e.g. prod, us-east"
              disabled={tagsMutation.isPending}
            />
            <p className="text-xs text-muted-foreground">
              Comma-separated. Members in this group get access to tenants whose
              tags are a subset of these tags.
            </p>
            <div className="flex justify-end">
              <Button
                size="sm"
                onClick={handleSaveTags}
                disabled={tagsMutation.isPending}
              >
                {tagsMutation.isPending ? 'Saving...' : 'Save Tags'}
              </Button>
            </div>
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
                      onClick={() => removeMemberMutation.mutate(m.metadata.id)}
                      disabled={removeMemberMutation.isPending}
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
                  onValueChange={(v) => addMemberMutation.mutate(v)}
                  disabled={addMemberMutation.isPending}
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
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Done
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
