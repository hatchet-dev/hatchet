import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import {
  AcceptedTenantInfo,
  useInviteActions,
} from '@/hooks/use-invite-actions';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import { Tenant } from '@/lib/api';
import { useUserUniverse } from '@/providers/user-universe';
import { CheckIcon, Cross2Icon } from '@radix-ui/react-icons';
import { useEffect, useMemo, useState } from 'react';

interface InviteModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function InviteModal({ isOpen, onClose }: InviteModalProps) {
  const { pendingInvitesQuery, invalidate: invalidatePendingInvites } =
    usePendingInvites();
  const { tenantMemberships } = useUserUniverse();
  const { setTenant } = useTenantDetails();

  const [phase, setPhase] = useState<'invites' | 'confirmation'>('invites');

  const tenantInvites = pendingInvitesQuery.data?.tenantInvites ?? [];
  const organizationInvites =
    pendingInvitesQuery.data?.organizationInvites ?? [];
  const totalInviteCount = pendingInvitesQuery.data?.inviteCount ?? 0;

  const {
    pendingId,
    processedIds,
    acceptedTenantInfos,
    errors,
    handleTenantAccept,
    handleTenantReject,
    handleOrgAccept,
    handleOrgReject,
    reset,
  } = useInviteActions({
    tenantInvites,
    organizationInvites,
    invalidatePendingInvites,
    onClose,
    onConfirmation: () => setPhase('confirmation'),
  });

  // Reset all action state when the modal closes
  useEffect(() => {
    if (!isOpen) {
      invalidatePendingInvites();
      setPhase('invites');
      reset();
    }
  }, [isOpen, invalidatePendingInvites, reset]);

  // Close immediately if opened with stale data that resolves to 0 invites
  useEffect(() => {
    if (
      isOpen &&
      phase === 'invites' &&
      pendingInvitesQuery.isSuccess &&
      totalInviteCount === 0 &&
      processedIds.size === 0
    ) {
      onClose();
    }
  }, [
    isOpen,
    phase,
    pendingInvitesQuery.isSuccess,
    totalInviteCount,
    processedIds.size,
    onClose,
  ]);

  const visibleTenantInvites = tenantInvites.filter(
    (inv) => !processedIds.has(inv.metadata.id),
  );
  const visibleOrgInvites = organizationInvites.filter(
    (inv) => !processedIds.has(inv.metadata.id),
  );

  const acceptedTenants = useMemo(
    () =>
      acceptedTenantInfos
        .map((info) => ({
          info,
          tenant: tenantMemberships?.find(
            (m) => m.tenant?.metadata.id === info.id,
          )?.tenant,
        }))
        .filter((e): e is { info: AcceptedTenantInfo; tenant: Tenant } =>
          Boolean(e.tenant),
        ),
    [acceptedTenantInfos, tenantMemberships],
  );

  const pendingCount = visibleTenantInvites.length + visibleOrgInvites.length;

  const titleText =
    pendingCount > 1
      ? `${pendingCount} pending invites`
      : visibleTenantInvites[0]
        ? `Join ${visibleTenantInvites[0].tenantName ?? 'a tenant'}`
        : visibleOrgInvites[0]
          ? `Join ${visibleOrgInvites[0].organizationName ?? 'an organization'}`
          : 'Pending invite';

  // Suppress content render while the stale-close effect is pending to avoid flash
  const isStaleClose =
    isOpen &&
    phase === 'invites' &&
    pendingInvitesQuery.isSuccess &&
    totalInviteCount === 0 &&
    processedIds.size === 0;

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(o) => {
        if (!o) {
          onClose();
        }
      }}
    >
      <DialogContent
        className="max-w-2xl"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        {!isStaleClose && (
          <>
            {phase === 'invites' ? (
              <>
                <DialogHeader>
                  <DialogTitle>{titleText}</DialogTitle>
                  <DialogDescription>
                    {pendingCount === 1
                      ? 'Accept or decline the invite below.'
                      : `You have ${pendingCount} pending invite${pendingCount !== 1 ? 's' : ''}.`}
                  </DialogDescription>
                </DialogHeader>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">Type</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead className="w-24">Role</TableHead>
                      <TableHead className="w-44">From</TableHead>
                      <TableHead className="w-20 text-right" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {visibleTenantInvites.map((invite) => (
                      <TableRow key={invite.metadata.id}>
                        <TableCell>
                          <Badge
                            variant="secondary"
                            className="text-xs font-normal"
                          >
                            Tenant
                          </Badge>
                        </TableCell>
                        <TableCell className="font-medium">
                          {invite.tenantName ?? '—'}
                        </TableCell>
                        <TableCell className="capitalize text-muted-foreground">
                          {invite.role.toLowerCase()}
                        </TableCell>
                        <TableCell
                          className="max-w-[176px] truncate text-xs text-muted-foreground"
                          title={invite.email}
                        >
                          {invite.email}
                        </TableCell>
                        <TableCell className="text-right">
                          <InviteActions
                            disabled={pendingId === invite.metadata.id}
                            onAccept={() =>
                              handleTenantAccept(
                                invite.metadata.id,
                                invite.tenantId,
                                invite.tenantName ?? '',
                              )
                            }
                            onDecline={() =>
                              handleTenantReject(invite.metadata.id)
                            }
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                    {visibleOrgInvites.map((invite) => (
                      <TableRow key={invite.metadata.id}>
                        <TableCell>
                          <Badge
                            variant="secondary"
                            className="text-xs font-normal"
                          >
                            Org
                          </Badge>
                        </TableCell>
                        <TableCell className="font-medium">
                          {invite.organizationName ?? '—'}
                        </TableCell>
                        <TableCell className="capitalize text-muted-foreground">
                          {invite.role.toLowerCase()}
                        </TableCell>
                        <TableCell
                          className="max-w-[176px] truncate text-xs text-muted-foreground"
                          title={invite.inviterEmail}
                        >
                          {invite.inviterEmail}
                        </TableCell>
                        <TableCell className="text-right">
                          <InviteActions
                            disabled={pendingId === invite.metadata.id}
                            onAccept={() => handleOrgAccept(invite.metadata.id)}
                            onDecline={() =>
                              handleOrgReject(invite.metadata.id)
                            }
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </>
            ) : (
              <ConfirmationStep
                acceptedTenantInfos={acceptedTenantInfos}
                acceptedTenants={acceptedTenants}
                onSwitch={(tenant) => {
                  setTenant(tenant);
                  onClose();
                }}
                onClose={onClose}
              />
            )}
            {errors.length > 0 && (
              <Alert variant="destructive">
                <AlertDescription>
                  {errors.map((e, i) => (
                    <p key={i}>{e}</p>
                  ))}
                </AlertDescription>
              </Alert>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function ConfirmationStep({
  acceptedTenantInfos,
  acceptedTenants,
  onSwitch,
  onClose,
}: {
  acceptedTenantInfos: AcceptedTenantInfo[];
  acceptedTenants: Array<{ info: AcceptedTenantInfo; tenant: Tenant }>;
  onSwitch: (tenant: Tenant) => void;
  onClose: () => void;
}) {
  const stillLoading =
    acceptedTenantInfos.length > 0 && acceptedTenants.length === 0;
  const hasTenants = acceptedTenants.length > 0;

  return (
    <>
      <DialogHeader>
        <DialogTitle>You&apos;re all set!</DialogTitle>
        <DialogDescription>
          {!hasTenants && !stillLoading
            ? 'Your invites have been processed.'
            : hasTenants && acceptedTenants.length === 1
              ? `You joined ${acceptedTenants[0].info.name}. Switch to it now?`
              : hasTenants
                ? `You joined ${acceptedTenants.length} new tenants. Select one to switch to.`
                : 'Processing…'}
        </DialogDescription>
      </DialogHeader>

      <div className="flex flex-col gap-3">
        {!hasTenants && !stillLoading && (
          <div className="flex justify-end">
            <Button onClick={onClose}>Done</Button>
          </div>
        )}

        {stillLoading && (
          <>
            {acceptedTenantInfos.map((info) => (
              <div
                key={info.id}
                className="flex items-center justify-between rounded-md border p-3"
              >
                <span className="font-medium">{info.name}</span>
                <Button size="sm" disabled>
                  Loading…
                </Button>
              </div>
            ))}
            <div className="flex justify-end">
              <Button variant="outline" onClick={onClose}>
                Maybe later
              </Button>
            </div>
          </>
        )}

        {hasTenants && acceptedTenants.length === 1 && (
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={onClose}>
              Maybe later
            </Button>
            <Button onClick={() => onSwitch(acceptedTenants[0].tenant)}>
              Switch to {acceptedTenants[0].info.name}
            </Button>
          </div>
        )}

        {hasTenants && acceptedTenants.length > 1 && (
          <>
            {acceptedTenants.map(({ info, tenant }) => (
              <div
                key={info.id}
                className="flex items-center justify-between rounded-md border p-3"
              >
                <span className="font-medium">{info.name}</span>
                <Button size="sm" onClick={() => onSwitch(tenant)}>
                  Switch
                </Button>
              </div>
            ))}
            <div className="flex justify-end">
              <Button variant="outline" onClick={onClose}>
                Maybe later
              </Button>
            </div>
          </>
        )}
      </div>
    </>
  );
}

function InviteActions({
  disabled,
  onAccept,
  onDecline,
}: {
  disabled: boolean;
  onAccept: () => void;
  onDecline: () => void;
}) {
  return (
    <div className="flex justify-end gap-1">
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-muted-foreground hover:text-foreground"
        disabled={disabled}
        onClick={onDecline}
        hoverText="Decline"
        aria-label="Decline"
      >
        <Cross2Icon className="h-4 w-4" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-muted-foreground hover:text-green-500"
        disabled={disabled}
        onClick={onAccept}
        hoverText="Accept"
        aria-label="Accept"
      >
        <CheckIcon className="h-4 w-4" />
      </Button>
    </div>
  );
}
