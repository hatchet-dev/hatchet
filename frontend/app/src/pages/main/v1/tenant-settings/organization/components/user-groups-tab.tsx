import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import {
  OrganizationMember,
  UserGroup,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { TagBadge } from '@/pages/main/v1/tenant-settings/organization/components/tag-badge';
import { CreateUserGroupModal } from '@/pages/organizations/$organization/components/create-user-group-modal';
import { EditUserGroupModal } from '@/pages/organizations/$organization/components/edit-user-group-modal';
import {
  EllipsisVerticalIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { useState } from 'react';

interface UserGroupsTabProps {
  organizationId: string;
  allOrgMembers: OrganizationMember[];
}

export function UserGroupsTab({
  organizationId,
  allOrgMembers,
}: UserGroupsTabProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});

  const [showCreate, setShowCreate] = useState(false);
  const [groupToEdit, setGroupToEdit] = useState<UserGroup | null>(null);

  const groupsQuery = useQuery({
    ...orgApi.userGroupsListQuery(organizationId),
  });

  const groups = groupsQuery.data?.rows ?? [];

  const columns = [
    {
      columnLabel: 'Name',
      cellRenderer: (row: UserGroup) => (
        <span className="font-medium">{row.name}</span>
      ),
    },
    {
      columnLabel: 'Tenant Role',
      cellRenderer: (row: UserGroup) => (
        <Badge variant="outline">{row.role}</Badge>
      ),
    },
    {
      columnLabel: 'Tags',
      cellRenderer: (row: UserGroup) =>
        row.tags.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {row.tags.map((tag) => (
              <TagBadge key={tag} tag={tag} />
            ))}
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
    },
    {
      columnLabel: 'Members',
      cellRenderer: (row: UserGroup) => (
        <span className="text-sm">{row.memberCount}</span>
      ),
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (row: UserGroup) => (
        <UserGroupActions
          group={row}
          organizationId={organizationId}
          onEdit={setGroupToEdit}
          onDeleted={() =>
            queryClient.invalidateQueries({
              queryKey: ['organization:user-groups:list', organizationId],
            })
          }
          onError={handleApiError}
        />
      ),
    },
  ];

  return (
    <>
      <div className="mb-4 flex justify-end">
        <Button onClick={() => setShowCreate(true)}>New Group</Button>
      </div>

      {groupsQuery.isLoading ? (
        <div className="py-8 text-center text-sm text-muted-foreground">
          Loading...
        </div>
      ) : groups.length > 0 ? (
        <SimpleTable
          data={groups}
          columns={columns}
          rowKey={(row) => row.metadata.id}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No user groups yet.
        </div>
      )}

      <CreateUserGroupModal
        open={showCreate}
        onOpenChange={setShowCreate}
        organizationId={organizationId}
      />

      {groupToEdit && (
        <EditUserGroupModal
          open={!!groupToEdit}
          onOpenChange={(open) => !open && setGroupToEdit(null)}
          organizationId={organizationId}
          group={groupToEdit}
          allOrgMembers={allOrgMembers}
        />
      )}
    </>
  );
}

function UserGroupActions({
  group,
  organizationId,
  onEdit,
  onDeleted,
  onError,
}: {
  group: UserGroup;
  organizationId: string;
  onEdit: (group: UserGroup) => void;
  onDeleted: () => void;
  onError: (error: AxiosError) => void;
}) {
  const orgApi = useOrganizationApi();
  const deleteMutation = useMutation({
    ...orgApi.userGroupDeleteMutation(organizationId, group.metadata.id),
    onSuccess: onDeleted,
    onError,
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
          <EllipsisVerticalIcon className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => onEdit(group)}>
          <PencilSquareIcon className="mr-2 size-4" />
          Edit
        </DropdownMenuItem>
        <DropdownMenuItem
          className="text-destructive"
          onClick={() => deleteMutation.mutate()}
          disabled={deleteMutation.isPending}
        >
          <TrashIcon className="mr-2 size-4" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
