import { APICloudMetadata } from '@/lib/api/generated/cloud/data-contracts';
import { Tenant } from '@/lib/api/generated/data-contracts';
import { BillingContext } from '@/lib/atoms';

type EvaluateContext = {
  tenant?: Tenant;
  billing?: BillingContext;
  meta?: APICloudMetadata;
};

export type Evaluate = (
  context: EvaluateContext,
) => [boolean, RejectReason | undefined];

export type PermissionSet<K = unknown> = Record<
  string,
  (resource?: K) => Evaluate
>;

// Allow different resource types per permission
export type FlexiblePermissionSet = Record<
  string,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (...args: any[]) => Evaluate
>;

export enum RejectReason {
  BILLING_REQUIRED = 'BILLING_REQUIRED',
  UPGRADE_REQUIRED = 'UPGRADE_REQUIRED',
}
