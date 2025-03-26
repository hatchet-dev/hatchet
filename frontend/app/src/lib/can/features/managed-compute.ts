import {
  Evaluate,
  EvaluateContext,
  RejectReason,
} from '../shared/permission.base';

export const managedCompute: Record<string, Evaluate> = {
  create: (context: EvaluateContext) => {
    const requireBillingForManagedCompute =
      context.meta?.requireBillingForManagedCompute;

    if (
      requireBillingForManagedCompute &&
      !context.billing?.hasPaymentMethods
    ) {
      return [false, RejectReason.BILLING_REQUIRED];
    }

    return [true, undefined];
  },
};
