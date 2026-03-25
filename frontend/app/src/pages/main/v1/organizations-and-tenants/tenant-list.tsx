import { formatInviteExpiry } from "./format-invite-expiry";
import type { TenantWithRole } from "./index";
import { Button } from "@/components/v1/ui/button";
import CopyToClipboard from "@/components/v1/ui/copy-to-clipboard";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/v1/ui/table";
import {
  TenantInvite,
  TenantMember,
  TenantMemberRole,
} from "@/lib/api/generated/data-contracts";
import { globalEmitter } from "@/lib/global-emitter";
import { capitalize } from "@/lib/utils";
import { appRoutes } from "@/router";
import { ChevronRightIcon, PlusIcon } from "@heroicons/react/24/outline";
import { Link } from "@tanstack/react-router";
import { Fragment, useState } from "react";

export const TenantTable = ({
  tenants,
  tenantMembers,
  tenantInvites,
  onInviteMember,
}: {
  tenants: TenantWithRole[];
  tenantMembers: Map<string, null | TenantMember[]>;
  tenantInvites: Map<string, null | TenantInvite[]>;
  onInviteMember: (tenantId: string) => void;
}) => {
  const [expandedTenants, setExpandedTenants] = useState<Set<string>>(
    new Set(),
  );

  const toggleExpanded = (tenantId: string) => {
    setExpandedTenants((prev) => {
      const next = new Set(prev);
      if (next.has(tenantId)) {
        next.delete(tenantId);
      } else {
        next.add(tenantId);
      }
      return next;
    });
  };

  return (
    <div className="overflow-hidden rounded-md border bg-background">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>ID</TableHead>
            <TableHead>Slug</TableHead>
            <TableHead>Members</TableHead>
            <TableHead>Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {tenants.map((tenant) => {
            const canManage =
              tenant.currentUsersRole === TenantMemberRole.OWNER ||
              tenant.currentUsersRole === TenantMemberRole.ADMIN;
            const members = tenantMembers.get(tenant.metadata.id);
            const invites = tenantInvites.get(tenant.metadata.id) ?? [];
            const isExpanded = expandedTenants.has(tenant.metadata.id);

            return (
              <Fragment key={tenant.metadata.id}>
                <TableRow>
                  <TableCell>
                    <Link
                      to={appRoutes.tenantRoute.to}
                      params={{ tenant: tenant.metadata.id }}
                      className="font-medium hover:underline"
                    >
                      {tenant.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-sm">
                        {tenant.metadata.id}
                      </span>
                      <CopyToClipboard text={tenant.metadata.id} />
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-muted-foreground">{tenant.slug}</span>
                  </TableCell>
                  <TableCell>
                    {members != null ? (
                      <button
                        onClick={() => toggleExpanded(tenant.metadata.id)}
                        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
                      >
                        <ChevronRightIcon
                          className={`size-3.5 transition-transform ${isExpanded ? "rotate-90" : ""}`}
                        />
                        {members.length}
                      </button>
                    ) : null}
                  </TableCell>
                  <TableCell>
                    {canManage && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onInviteMember(tenant.metadata.id)}
                        leftIcon={<PlusIcon className="size-4" />}
                      >
                        Invite new member
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
                {isExpanded &&
                  members?.map((member) => (
                    <TableRow
                      key={`${tenant.metadata.id}-${member.metadata.id}`}
                      className="bg-muted/30"
                    >
                      <TableCell>
                        <span className="text-sm">
                          {member.user?.email ?? "-"}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm font-medium">
                          {capitalize(member.role)}
                        </span>
                      </TableCell>
                      <TableCell />
                      <TableCell />
                      <TableCell />
                    </TableRow>
                  ))}
                {isExpanded &&
                  invites.map((invite) => (
                    <TableRow
                      key={`${tenant.metadata.id}-invite-${invite.metadata.id}`}
                      className="bg-muted/30 text-muted-foreground"
                    >
                      <TableCell>
                        <span className="text-sm">{invite.email}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          Invited {formatInviteExpiry(invite.expires)}
                        </span>
                      </TableCell>
                      <TableCell />
                      <TableCell />
                      <TableCell />
                    </TableRow>
                  ))}
              </Fragment>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
};

export const TenantList = ({
  tenants,
  tenantMembers,
  tenantInvites,
}: {
  tenants: TenantWithRole[];
  tenantMembers: Map<string, null | TenantMember[]>;
  tenantInvites: Map<string, null | TenantInvite[]>;
}) => {
  if (tenants.length === 0) {
    return (
      <div className="py-16 text-center">
        <h3 className="mb-2 text-lg font-medium">No Tenants</h3>
        <p className="text-muted-foreground">
          You are not a member of any tenants.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Tenants</h2>
        <Button
          variant="outline"
          size="sm"
          onClick={() => globalEmitter.emit("create-new-tenant", {})}
          leftIcon={<PlusIcon className="size-4" />}
        >
          Add a tenant
        </Button>
      </div>
      <TenantTable
        tenants={tenants}
        tenantMembers={tenantMembers}
        tenantInvites={tenantInvites}
        onInviteMember={(tenantId) =>
          globalEmitter.emit("create-tenant-invite", { tenantId })
        }
      />
    </div>
  );
};
