import { useAnalytics } from '@/hooks/use-analytics';
import { useOrganizations } from '@/hooks/use-organizations';
import { TenantInvite } from '@/lib/api';
import { OrganizationInvite } from '@/lib/api/generated/cloud/data-contracts';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { useMutation } from '@tanstack/react-query';
import { useCallback, useRef, useState } from 'react';

export interface AcceptedTenantInfo {
  id: string;
  name: string;
}

export function useInviteActions({
  tenantInvites,
  organizationInvites,
  invalidatePendingInvites,
  onClose,
  onConfirmation,
}: {
  tenantInvites: TenantInvite[];
  organizationInvites: OrganizationInvite[];
  invalidatePendingInvites: () => void;
  onClose: () => void;
  onConfirmation: () => void;
}) {
  const { invalidate: invalidateUserUniverse } = useUserUniverse();
  const { acceptOrgInviteMutation, rejectOrgInviteMutation } =
    useOrganizations();
  const { tenantInviteAcceptMutation, tenantInviteRejectMutation } =
    useTenantApi();
  const { capture } = useAnalytics();

  const [pendingId, setPendingId] = useState<string>();
  const [processedIds, setProcessedIds] = useState<Set<string>>(new Set());
  const [acceptedTenantInfos, setAcceptedTenantInfos] = useState<
    AcceptedTenantInfo[]
  >([]);
  const [errors, setErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors });

  // Refs give synchronous access to current values inside mutation callbacks,
  // since React state updates are deferred until the next render.
  const processedIdsRef = useRef<Set<string>>(new Set());
  const acceptedTenantInfosRef = useRef<AcceptedTenantInfo[]>([]);

  const checkCompletion = useCallback(
    (nextProcessedIds: Set<string>, nextAccepted: AcceptedTenantInfo[]) => {
      const remaining =
        tenantInvites.filter((inv) => !nextProcessedIds.has(inv.metadata.id))
          .length +
        organizationInvites.filter(
          (inv) => !nextProcessedIds.has(inv.metadata.id),
        ).length;
      if (remaining > 0) {
        return;
      }
      invalidatePendingInvites();
      if (nextAccepted.length > 0) {
        onConfirmation();
      } else {
        onClose();
      }
    },
    [
      tenantInvites,
      organizationInvites,
      invalidatePendingInvites,
      onClose,
      onConfirmation,
    ],
  );

  const markProcessed = useCallback((id: string) => {
    const next = new Set([...processedIdsRef.current, id]);
    processedIdsRef.current = next;
    setProcessedIds(next);
    return next;
  }, []);

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
    onSuccess: ({ tenantId, tenantName }) => {
      const next = [
        ...acceptedTenantInfosRef.current,
        { id: tenantId, name: tenantName },
      ];
      acceptedTenantInfosRef.current = next;
      setAcceptedTenantInfos(next);
      // Deliberately not awaited: closing the modal must not wait on the
      // memberships refetch. The confirmation step shows a loading state
      // until the new tenant appears.
      void invalidateUserUniverse();
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
            const nextProcessedIds = markProcessed(inviteId);
            invalidatePendingInvites();
            checkCompletion(nextProcessedIds, acceptedTenantInfosRef.current);
          },
          onSettled: () => setPendingId(undefined),
        },
      );
    },
    [
      acceptTenantMutation,
      capture,
      markProcessed,
      invalidatePendingInvites,
      checkCompletion,
    ],
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
            const nextProcessedIds = markProcessed(inviteId);
            invalidatePendingInvites();
            checkCompletion(nextProcessedIds, acceptedTenantInfosRef.current);
          },
          onSettled: () => setPendingId(undefined),
        },
      );
    },
    [
      rejectTenantMutation,
      capture,
      markProcessed,
      invalidatePendingInvites,
      checkCompletion,
    ],
  );

  const handleOrgAccept = useCallback(
    (inviteId: string) => {
      setErrors([]);
      setPendingId(inviteId);
      acceptOrgInviteMutation.mutate(
        { inviteId },
        {
          onSuccess: () => {
            // Deliberately not awaited: closing the modal must not wait on
            // the memberships refetch.
            void invalidateUserUniverse();
            capture('onboarding_org_invite_accepted', { invite_id: inviteId });
            const nextProcessedIds = markProcessed(inviteId);
            invalidatePendingInvites();
            checkCompletion(nextProcessedIds, acceptedTenantInfosRef.current);
          },
          onError: handleApiError,
          onSettled: () => setPendingId(undefined),
        },
      );
    },
    [
      acceptOrgInviteMutation,
      invalidateUserUniverse,
      capture,
      markProcessed,
      invalidatePendingInvites,
      checkCompletion,
      handleApiError,
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
            const nextProcessedIds = markProcessed(inviteId);
            invalidatePendingInvites();
            checkCompletion(nextProcessedIds, acceptedTenantInfosRef.current);
          },
          onError: handleApiError,
          onSettled: () => setPendingId(undefined),
        },
      );
    },
    [
      rejectOrgInviteMutation,
      capture,
      markProcessed,
      invalidatePendingInvites,
      checkCompletion,
      handleApiError,
    ],
  );

  const reset = useCallback(() => {
    processedIdsRef.current = new Set();
    acceptedTenantInfosRef.current = [];
    setPendingId(undefined);
    setProcessedIds(new Set());
    setAcceptedTenantInfos([]);
    setErrors([]);
  }, []);

  return {
    pendingId,
    processedIds,
    acceptedTenantInfos,
    errors,
    handleTenantAccept,
    handleTenantReject,
    handleOrgAccept,
    handleOrgReject,
    reset,
  };
}
