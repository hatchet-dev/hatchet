import { CancelInviteModal } from './cancel-invite-modal';
import { DeleteMemberModal } from './delete-member-modal';
import { InviteMemberModal } from './invite-member-modal';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Loading } from '@/components/v1/ui/loading';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useCurrentUser } from '@/hooks/use-current-user';
import { cloudApi } from '@/lib/api/api';
import {
  Organization,
  OrganizationInvite,
  OrganizationInviteStatus,
  OrganizationMember,
} from '@/lib/api/generated/cloud/data-contracts';
import { PlusIcon, UserIcon, EnvelopeIcon } from '@heroicons/react/24/outline';
import { EllipsisVerticalIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';
import { useState } from 'react';

interface MembersTabProps {
  organization: Organization;
  orgId: string;
  onRefetch: () => void;
}

export function MembersTab({
  organization,
  orgId,
  onRefetch,
}: MembersTabProps) {
  const { currentUser } = useCurrentUser();
  const [showInviteMemberModal, setShowInviteMemberModal] = useState(false);
  const [memberToDelete, setMemberToDelete] =
    useState<OrganizationMember | null>(null);
  const [inviteToCancel, setInviteToCancel] =
    useState<OrganizationInvite | null>(null);

  const organizationInvitesQuery = useQuery({
    queryKey: ['organization-invites:list', orgId],
    queryFn: async () => {
      const result = await cloudApi.organizationInviteList(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  const formatExpirationDate = (expiresDate: string) => {
    try {
      const expires = new Date(expiresDate);
      const now = new Date();
      if (expires < now) {
        return 'expired';
      }
      return `in ${formatDistanceToNow(expires)}`;
    } catch {
      return new Date(expiresDate).toLocaleDateString();
    }
  };

  return (
    <div className="space-y-8">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium">Members</h3>
            <p className="text-sm text-muted-foreground">
              Members with access to this organization
            </p>
          </div>
        </div>

        {organization.members && organization.members.length > 0 ? (
          <div className="space-y-4">
            <div className="hidden md:block">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {organization.members.map((member) => (
                    <TableRow key={member.metadata.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm">
                            {member.metadata.id}
                          </span>
                          <CopyToClipboard text={member.metadata.id} />
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {member.email}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{member.role}</Badge>
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0"
                            >
                              <EllipsisVerticalIcon className="size-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            {currentUser?.email === member.email ? (
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <DropdownMenuItem
                                      disabled
                                      className="cursor-not-allowed text-gray-400"
                                    >
                                      <TrashIcon className="mr-2 size-4" />
                                      Remove Member
                                    </DropdownMenuItem>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    <p>Cannot remove yourself</p>
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            ) : (
                              <DropdownMenuItem
                                onClick={() => setMemberToDelete(member)}
                              >
                                <TrashIcon className="mr-2 size-4" />
                                Remove Member
                              </DropdownMenuItem>
                            )}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <div className="space-y-4 md:hidden">
              {organization.members.map((member) => (
                <div
                  key={member.metadata.id}
                  className="space-y-3 rounded-lg border p-4"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-sm">{member.email}</span>
                      <Badge variant="default">{member.role}</Badge>
                    </div>
                    {currentUser?.email !== member.email && (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                          >
                            <EllipsisVerticalIcon className="size-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => setMemberToDelete(member)}
                          >
                            <TrashIcon className="mr-2 size-4" />
                            Remove Member
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </div>
                  <div className="space-y-2 text-sm">
                    <div>
                      <span className="font-medium text-muted-foreground">
                        Member ID:
                      </span>
                      <div className="mt-1 flex items-center gap-2">
                        <span className="font-mono text-sm">
                          {member.metadata.id}
                        </span>
                        <CopyToClipboard text={member.metadata.id} />
                      </div>
                    </div>
                    <div>
                      <span className="font-medium text-muted-foreground">
                        Member Since:
                      </span>
                      <span className="ml-2">
                        {new Date(
                          member.metadata.createdAt,
                        ).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="py-8 text-center">
            <UserIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
            <h3 className="mb-2 text-lg font-medium">No Members Yet</h3>
            <p className="mb-4 text-muted-foreground">
              Members will appear here when they join this organization.
            </p>
          </div>
        )}
      </div>

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium">Invites</h3>
            <p className="text-sm text-muted-foreground">
              Pending invitations to join this organization
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowInviteMemberModal(true)}
            leftIcon={<PlusIcon className="size-4" />}
          >
            Invite Member
          </Button>
        </div>

        {organizationInvitesQuery.isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loading />
          </div>
        ) : organizationInvitesQuery.data?.rows &&
          organizationInvitesQuery.data.rows.length > 0 ? (
          <div className="space-y-4">
            <div className="hidden md:block">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Expiry</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {organizationInvitesQuery.data.rows
                    .filter(
                      (invite) =>
                        invite.status === OrganizationInviteStatus.PENDING ||
                        invite.status === OrganizationInviteStatus.EXPIRED,
                    )
                    .map((invite) => (
                      <TableRow key={invite.metadata.id}>
                        <TableCell className="font-mono text-sm">
                          {invite.inviteeEmail}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{invite.role}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={
                              invite.status === OrganizationInviteStatus.PENDING
                                ? 'secondary'
                                : invite.status ===
                                    OrganizationInviteStatus.ACCEPTED
                                  ? 'default'
                                  : 'destructive'
                            }
                          >
                            {invite.status}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {formatExpirationDate(invite.expires)}
                        </TableCell>
                        <TableCell>
                          {invite.status ===
                            OrganizationInviteStatus.PENDING && (
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-8 w-8 p-0"
                                >
                                  <EllipsisVerticalIcon className="size-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onClick={() => setInviteToCancel(invite)}
                                >
                                  <TrashIcon className="mr-2 size-4" />
                                  Cancel Invitation
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </div>

            <div className="space-y-4 md:hidden">
              {organizationInvitesQuery.data.rows.map((invite) => (
                <div
                  key={invite.metadata.id}
                  className="space-y-3 rounded-lg border p-4"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-sm">
                        {invite.inviteeEmail}
                      </span>
                      <Badge variant="outline">{invite.role}</Badge>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge
                        variant={
                          invite.status === OrganizationInviteStatus.PENDING
                            ? 'secondary'
                            : invite.status ===
                                OrganizationInviteStatus.ACCEPTED
                              ? 'default'
                              : 'destructive'
                        }
                      >
                        {invite.status}
                      </Badge>
                      {invite.status === OrganizationInviteStatus.PENDING && (
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0"
                            >
                              <EllipsisVerticalIcon className="size-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              onClick={() => setInviteToCancel(invite)}
                            >
                              <TrashIcon className="mr-2 size-4" />
                              Cancel Invitation
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      )}
                    </div>
                  </div>
                  <div className="space-y-2 text-sm">
                    <div>
                      <span className="font-medium text-muted-foreground">
                        Invite ID:
                      </span>
                      <div className="mt-1 flex items-center gap-2">
                        <span className="font-mono text-sm">
                          {invite.metadata.id}
                        </span>
                        <CopyToClipboard text={invite.metadata.id} />
                      </div>
                    </div>
                    <div>
                      <span className="font-medium text-muted-foreground">
                        Invited By:
                      </span>
                      <span className="ml-2">{invite.inviterEmail}</span>
                    </div>
                    <div>
                      <span className="font-medium text-muted-foreground">
                        Expires:
                      </span>
                      <span className="ml-2">
                        {formatExpirationDate(invite.expires)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="py-8 text-center">
            <EnvelopeIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
            <h3 className="mb-2 text-lg font-medium">No Pending Invites</h3>
            <p className="mb-4 text-muted-foreground">
              Invite members to join this organization.
            </p>
            <Button
              onClick={() => setShowInviteMemberModal(true)}
              leftIcon={<PlusIcon className="size-4" />}
            >
              Invite Member
            </Button>
          </div>
        )}
      </div>

      <InviteMemberModal
        open={showInviteMemberModal}
        onOpenChange={setShowInviteMemberModal}
        organizationId={orgId}
        organizationName={organization.name}
        onSuccess={() => {
          onRefetch();
          organizationInvitesQuery.refetch();
        }}
      />

      {memberToDelete && (
        <DeleteMemberModal
          open={!!memberToDelete}
          onOpenChange={(open) => !open && setMemberToDelete(null)}
          member={memberToDelete}
          organizationName={organization.name}
          onSuccess={onRefetch}
        />
      )}

      {inviteToCancel && (
        <CancelInviteModal
          open={!!inviteToCancel}
          onOpenChange={(open) => !open && setInviteToCancel(null)}
          invite={inviteToCancel}
          organizationName={organization.name}
          onSuccess={() => organizationInvitesQuery.refetch()}
        />
      )}
    </div>
  );
}
