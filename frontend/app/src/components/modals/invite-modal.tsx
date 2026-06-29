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
import { useAnalytics } from '@/hooks/use-analytics';
import { useOrganizations } from '@/hooks/use-organizations';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import { Tenant } from '@/lib/api';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { CheckIcon, Cross2Icon } from '@radix-ui/react-icons';
import { useMutation } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useState } from 'react';

interface AcceptedTenantInfo {
  id: string;
  name: string;
}

interface InviteModalProps {
  open: boolean;
  onClose: () => void;
}

export function InviteModal({ open, onClose }: InviteModalProps) {
  const { pendingInvitesQuery, invalidate: invalidatePendingInvites } =
    usePendingInvites();
  const { invalidate: invalidateUserUniverse, tenantMemberships } =
    useUserUniverse();
  const { acceptOrgInviteMutation, rejectOrgInviteMutation } =
    useOrganizations();
  const { tenantInviteAcceptMutation, tenantInviteRejectMutation } =
    useTenantApi();
  const { capture } = useAnalytics();
  const { setTenant } = useTenantDetails();

  const [processedIds, setProcessedIds] = useState<Set<string>>(new Set());
  const [acceptedTenantInfos, setAcceptedTenantInfos] = useState<
    AcceptedTenantInfo[]
  >([]);
  const [phase, setPhase] = useState<'invites' | 'confirmation'>('invites');
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [errors, setErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors });

  // Reset state when modal closes and refresh invite count for the notification bar
  useEffect(() => {
    if (!open) {
      invalidatePendingInvites();
      setProcessedIds(new Set());
      setAcceptedTenantInfos([]);
      setPhase('invites');
      setPendingId(null);
      setErrors([]);
    }
  }, [open, invalidatePendingInvites]);

  const tenantInvites = pendingInvitesQuery.data?.tenantInvites ?? [];
  const organizationInvites =
    pendingInvitesQuery.data?.organizationInvites ?? [];
  const totalInviteCount = pendingInvitesQuery.data?.inviteCount ?? 0;

  // Close immediately if the modal was opened with stale data and the query resolves to 0
  useEffect(() => {
    if (
      open &&
      phase === 'invites' &&
      pendingInvitesQuery.isSuccess &&
      totalInviteCount === 0 &&
      processedIds.size === 0
    ) {
      onClose();
    }
  }, [
    open,
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

  // Derives full Tenant objects from live memberships — populates once invalidation resolves
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

  const markProcessed = useCallback((id: string) => {
    setProcessedIds((prev) => new Set([...prev, id]));
  }, []);

  // Transition to confirmation or close when all visible invites are handled
  useEffect(() => {
    if (phase !== 'invites') {
      return;
    }
    if (!pendingInvitesQuery.isSuccess) {
      return;
    }
    if (totalInviteCount === 0 || processedIds.size === 0) {
      return;
    }

    const remaining = visibleTenantInvites.length + visibleOrgInvites.length;
    if (remaining > 0) {
      return;
    }

    invalidatePendingInvites();
    if (acceptedTenantInfos.length > 0) {
      setPhase('confirmation');
    } else {
      onClose();
    }
  }, [
    processedIds.size,
    visibleTenantInvites.length,
    visibleOrgInvites.length,
    phase,
    acceptedTenantInfos.length,
    totalInviteCount,
    invalidatePendingInvites,
    onClose,
    pendingInvitesQuery.isSuccess,
  ]);

  const { mutationFn: acceptTenantFn } = tenantInviteAcceptMutation();
  const { mutationFn: rejectTenantFn } = tenantInviteRejectMutation();

  const acceptTenantMutation = useMutation({
    mutationKey: ['invite-modal:tenant:accept'],
    mutationFn: async (data: {
      inviteId: string;
      tenantId: string;
      tenantName: string;
    }) => {
      await acceptTenantFn({ invite: data.inviteId });
      return { tenantId: data.tenantId, tenantName: data.tenantName };
    },
    onSuccess: async ({ tenantId, tenantName }) => {
      await invalidateUserUniverse();
      setAcceptedTenantInfos((prev) => [
        ...prev,
        { id: tenantId, name: tenantName },
      ]);
    },
    onError: handleApiError,
  });

  const rejectTenantMutation = useMutation({
    mutationKey: ['invite-modal:tenant:reject'],
    mutationFn: async (data: { inviteId: string }) => {
      await rejectTenantFn({ invite: data.inviteId });
    },
    onError: handleApiError,
  });

  const handleTenantAccept = useCallback(
    (inviteId: string, tenantId: string, tenantName: string) => {
      setErrors([]);
      setPendingId(inviteId);
      acceptTenantMutation.mutate(
        { inviteId, tenantId, tenantName },
        {
          onSuccess: () => {
            capture('onboarding_tenant_invite_accepted', {
              invite_id: inviteId,
              tenant_id: tenantId,
            });
            markProcessed(inviteId);
            invalidatePendingInvites();
          },
          onSettled: () => setPendingId(null),
        },
      );
    },
    [acceptTenantMutation, capture, markProcessed, invalidatePendingInvites],
  );

  const handleTenantReject = useCallback(
    (inviteId: string) => {
      setErrors([]);
      setPendingId(inviteId);
      rejectTenantMutation.mutate(
        { inviteId },
        {
          onSuccess: () => {
            capture('onboarding_tenant_invite_rejected', {
              invite_id: inviteId,
            });
            markProcessed(inviteId);
            invalidatePendingInvites();
          },
          onSettled: () => setPendingId(null),
        },
      );
    },
    [rejectTenantMutation, capture, markProcessed, invalidatePendingInvites],
  );

  const handleOrgAccept = useCallback(
    (inviteId: string) => {
      setErrors([]);
      setPendingId(inviteId);
      acceptOrgInviteMutation.mutate(
        { inviteId },
        {
          onSuccess: async () => {
            await invalidateUserUniverse();
            capture('onboarding_org_invite_accepted', { invite_id: inviteId });
            markProcessed(inviteId);
            invalidatePendingInvites();
          },
          onSettled: () => setPendingId(null),
        },
      );
    },
    [
      acceptOrgInviteMutation,
      invalidateUserUniverse,
      capture,
      markProcessed,
      invalidatePendingInvites,
    ],
  );

  const handleOrgReject = useCallback(
    (inviteId: string) => {
      setErrors([]);
      setPendingId(inviteId);
      rejectOrgInviteMutation.mutate(
        { inviteId },
        {
          onSuccess: () => {
            capture('onboarding_org_invite_rejected', { invite_id: inviteId });
            markProcessed(inviteId);
            invalidatePendingInvites();
          },
          onSettled: () => setPendingId(null),
        },
      );
    },
    [rejectOrgInviteMutation, capture, markProcessed, invalidatePendingInvites],
  );

  const pendingCount = visibleTenantInvites.length + visibleOrgInvites.length;

  const titleText =
    pendingCount > 1
      ? `${pendingCount} pending invites`
      : visibleTenantInvites[0]
        ? `Join ${visibleTenantInvites[0].tenantName ?? 'a tenant'}`
        : visibleOrgInvites[0]
          ? `Join ${visibleOrgInvites[0].organizationName}`
          : 'Pending invite';

  return (
    <Dialog
      open={open}
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
                        onDecline={() => handleTenantReject(invite.metadata.id)}
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
                      {invite.organizationName}
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
                        onDecline={() => handleOrgReject(invite.metadata.id)}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </>
        ) : (
          // Confirmation step
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
        {/* No tenant invites were accepted */}
        {!hasTenants && !stillLoading && (
          <div className="flex justify-end">
            <Button onClick={onClose}>Done</Button>
          </div>
        )}

        {/* Waiting for membership data to refresh */}
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

        {/* Single accepted tenant */}
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

        {/* Multiple accepted tenants */}
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
