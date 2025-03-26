import { PermissionSet, RejectReason } from '../shared/permission.base';

export const managedCompute: PermissionSet = {
  create: () => (context) => {
    const requireBillingForManagedCompute =
      context.meta?.requireBillingForManagedCompute;

    if (
      requireBillingForManagedCompute &&
      !context.billing?.hasPaymentMethods
    ) {
      return [false, RejectReason.BILLING_REQUIRED];
    }

    return [false, RejectReason.BILLING_REQUIRED];
  },
  selectCompute: () => {
    return () => {
      return [true, undefined];
    };
  },
};
