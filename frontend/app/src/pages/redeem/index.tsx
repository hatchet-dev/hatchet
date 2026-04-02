import { NewOrganizationSaverForm } from '@/components/forms/new-organization-saver-form';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Separator } from '@/components/v1/ui/separator';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { useOrganizations } from '@/hooks/use-organizations';
import { queries } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import {
  UserOffer,
  UserOfferList,
  UserOfferStage,
  UserOfferType,
} from '@/lib/api/generated/cloud/data-contracts';
import { lastTenantAtom } from '@/lib/atoms';
import { appRoutes } from '@/router';
import {
  ArrowLeftIcon,
  ExclamationTriangleIcon,
  GiftIcon,
  CheckCircleIcon,
  PlusIcon,
} from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { useAtomValue } from 'jotai';
import { useMemo, useState } from 'react';

function formatCurrency(cents: number) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(cents / 100);
}

function offerTypeBadgeVariant(
  type?: UserOfferType,
): 'default' | 'secondary' | 'outline' {
  switch (type) {
    case UserOfferType.YCAlumni:
    case UserOfferType.YCCurrentBatch:
      return 'default';
    case UserOfferType.Startup:
      return 'secondary';
    default:
      return 'outline';
  }
}

function stageBadgeVariant(
  stage?: UserOfferStage,
): 'successful' | 'inProgress' | 'queued' {
  switch (stage) {
    case UserOfferStage.Approved:
      return 'successful';
    case UserOfferStage.Redeemed:
      return 'inProgress';
    default:
      return 'queued';
  }
}

const CREATE_NEW_ORG_VALUE = '__create_new_org__';

function OfferCard({
  offer,
  organizations,
  defaultOrgId,
  onRedeem,
  onCreateOrg,
  isRedeeming,
  redeemingOfferId,
}: {
  offer: UserOffer;
  organizations: { metadata: { id: string }; name: string }[];
  defaultOrgId: string;
  onRedeem: (offerRecordId: string, organizationId: string) => void;
  onCreateOrg: () => void;
  isRedeeming: boolean;
  redeemingOfferId: string | null;
}) {
  const [selectedOrgId, setSelectedOrgId] = useState<string | null>(null);

  const effectiveOrgId = selectedOrgId ?? defaultOrgId;

  const handleOrgChange = (value: string) => {
    if (value === CREATE_NEW_ORG_VALUE) {
      onCreateOrg();
      return;
    }
    setSelectedOrgId(value);
  };

  const isRedeemable = offer.stage === UserOfferStage.Approved;
  const isRedeemed = offer.stage === UserOfferStage.Redeemed;
  const isThisRedeeming = isRedeeming && redeemingOfferId === offer.recordId;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {offer.type ? (
              <Badge variant={offerTypeBadgeVariant(offer.type)}>
                {offer.type}
              </Badge>
            ) : (
              <CardTitle className="text-base">Offer</CardTitle>
            )}
          </div>
          {offer.stage && (
            <Badge variant={stageBadgeVariant(offer.stage)}>
              {offer.stage}
            </Badge>
          )}
        </div>
        <CardDescription>
          {offer.creditAmountCents
            ? `${formatCurrency(offer.creditAmountCents)} credit`
            : 'Credit offer'}
          {offer.coupon && ` \u00b7 Coupon: ${offer.coupon}`}
          {offer.expiresAt &&
            ` \u00b7 Expires ${new Date(offer.expiresAt).toLocaleDateString()}`}
        </CardDescription>
      </CardHeader>
      {isRedeemable && (
        <CardContent>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="mb-1.5 block text-sm font-medium">
                Apply to organization
              </label>
              <Select value={effectiveOrgId} onValueChange={handleOrgChange}>
                <SelectTrigger>
                  <SelectValue placeholder="Select an organization" />
                </SelectTrigger>
                <SelectContent>
                  {organizations.map((org) => (
                    <SelectItem key={org.metadata.id} value={org.metadata.id}>
                      {org.name}
                    </SelectItem>
                  ))}
                  <Separator className="my-1" />
                  <SelectItem value={CREATE_NEW_ORG_VALUE}>
                    <span className="flex items-center gap-2">
                      <PlusIcon className="size-4" />
                      Create new organization
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Button
              onClick={() => onRedeem(offer.recordId, effectiveOrgId)}
              disabled={!effectiveOrgId || isRedeeming}
            >
              {isThisRedeeming ? (
                <>
                  <Spinner />
                  Redeeming...
                </>
              ) : (
                'Redeem'
              )}
            </Button>
          </div>
        </CardContent>
      )}
      {isRedeemed && (
        <CardContent>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircleIcon className="size-4 text-green-500" />
            This offer has been redeemed.
          </div>
        </CardContent>
      )}
    </Card>
  );
}

export default function RedeemOffersPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { organizations, isUserUniverseLoaded, getOrganizationIdForTenant } =
    useOrganizations();
  const lastTenant = useAtomValue(lastTenantAtom);
  const [redeemingOfferId, setRedeemingOfferId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const offersQuery = useQuery({
    ...queries.cloud.offers(),
  });

  const redeemMutation = useMutation({
    mutationKey: ['offer:redeem'],
    mutationFn: async (data: {
      offerRecordId: string;
      organizationId: string;
    }) => {
      const result = await cloudApi.userOfferRedeem({
        offerRecordId: data.offerRecordId,
        organizationId: data.organizationId,
      });
      return result.data;
    },
    onMutate: async (data) => {
      await queryClient.cancelQueries({ queryKey: ['offers:list'] });
      const previous = queryClient.getQueryData<UserOfferList>(['offers:list']);
      queryClient.setQueryData<UserOfferList>(['offers:list'], (old) => {
        if (!old) {
          return old;
        }
        return {
          ...old,
          rows: old.rows.map((o) =>
            o.recordId === data.offerRecordId
              ? { ...o, stage: UserOfferStage.Redeemed }
              : o,
          ),
        };
      });
      return { previous };
    },
    onSuccess: (data) => {
      setError(null);
      setSuccessMessage(
        `${formatCurrency(data.appliedCents)} in credit has been applied to your organization.`,
      );
      void queryClient.refetchQueries({ queryKey: ['offers:list'] });
      setRedeemingOfferId(null);
    },
    onError: (err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['offers:list'], context.previous);
      }
      setSuccessMessage(null);
      let message = 'An unexpected error occurred.';
      if (err instanceof AxiosError) {
        const errors = err.response?.data?.errors;
        if (Array.isArray(errors) && errors.length > 0) {
          message = errors
            .map(
              (e: Record<string, string>) => e.description || 'Unknown error',
            )
            .join(', ');
        }
      }
      setError(message);
      void queryClient.refetchQueries({ queryKey: ['offers:list'] });
      setRedeemingOfferId(null);
    },
  });

  const handleRedeem = (offerRecordId: string, organizationId: string) => {
    setError(null);
    setSuccessMessage(null);
    setRedeemingOfferId(offerRecordId);
    redeemMutation.mutate({ offerRecordId, organizationId });
  };

  const [createOrgOpen, setCreateOrgOpen] = useState(false);

  const isLoading = offersQuery.isLoading || !isUserUniverseLoaded;
  const offers = offersQuery.data?.rows ?? [];
  const defaultOrgId = useMemo(() => {
    if (lastTenant?.metadata.id) {
      const orgId = getOrganizationIdForTenant(lastTenant.metadata.id);
      if (orgId) {
        return orgId;
      }
    }
    return organizations.length > 0 ? organizations[0].metadata.id : '';
  }, [lastTenant?.metadata.id, getOrganizationIdForTenant, organizations]);

  const handleCreateOrg = () => {
    setCreateOrgOpen(true);
  };

  return (
    <div className="max-h-full overflow-y-auto">
      <div className="mx-auto max-w-2xl space-y-6 p-6">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate({ to: appRoutes.authenticatedRoute.to })}
            className="gap-2 text-muted-foreground"
          >
            <ArrowLeftIcon className="size-4" />
            Back to Dashboard
          </Button>
        </div>

        <div>
          <h1 className="text-2xl font-bold">Redeem Offers</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            View and redeem available offers for your account.
          </p>
        </div>

        {error && (
          <Alert variant="destructive">
            <ExclamationTriangleIcon className="size-4" />
            <AlertTitle>Redemption failed</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {successMessage && (
          <Alert variant="default" className="border-green-500/50">
            <CheckCircleIcon className="size-4 text-green-500" />
            <AlertTitle>Offer redeemed</AlertTitle>
            <AlertDescription>{successMessage}</AlertDescription>
          </Alert>
        )}

        {isLoading ? (
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <Skeleton className="h-5 w-28 rounded-md" />
                  <Skeleton className="h-5 w-20 rounded-md" />
                </div>
                <Skeleton className="mt-2 h-4 w-44" />
              </CardHeader>
              <CardContent>
                <div className="flex items-end gap-3">
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-4 w-36" />
                    <Skeleton className="h-9 w-full rounded-md" />
                  </div>
                  <Skeleton className="h-9 w-24 rounded-md" />
                </div>
              </CardContent>
            </Card>
          </div>
        ) : offers.length === 0 ? (
          <Card>
            <CardContent className="py-16 text-center">
              <GiftIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
              <h3 className="mb-2 text-lg font-medium">No Offers Available</h3>
              <p className="mb-4 text-muted-foreground">
                There are no offers associated with your account at this time.
              </p>
              <Button
                variant="outline"
                onClick={() =>
                  navigate({ to: appRoutes.authenticatedRoute.to })
                }
              >
                Go to Dashboard
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {offers.map((offer) => (
              <OfferCard
                key={offer.recordId}
                offer={offer}
                organizations={organizations}
                defaultOrgId={defaultOrgId}
                onRedeem={handleRedeem}
                onCreateOrg={handleCreateOrg}
                isRedeeming={redeemMutation.isPending}
                redeemingOfferId={redeemingOfferId}
              />
            ))}
          </div>
        )}
      </div>

      <Dialog open={createOrgOpen} onOpenChange={setCreateOrgOpen}>
        <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
          <DialogHeader>
            <DialogTitle>Create New Organization</DialogTitle>
          </DialogHeader>
          <div className="flex justify-center">
            <NewOrganizationSaverForm
              afterSave={() => {
                setCreateOrgOpen(false);
                queryClient.invalidateQueries({
                  queryKey: ['organization:list'],
                });
              }}
            />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
