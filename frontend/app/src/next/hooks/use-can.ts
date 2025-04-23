import {
  Evaluate,
  EvaluateResult,
} from '@/next/lib/can/shared/permission.base';

import useTenant from './use-tenant';
import useUser from './use-user';
import { useCallback } from 'react';
import useApiMeta from './use-api-meta';
import useBilling from './use-billing';

type Can = (evalFn: Evaluate) => boolean;
type CanWithReason = (evalFn: Evaluate) => EvaluateResult;

interface CanHook {
  can: Can;
  canWithReason: CanWithReason;
}

export default function useCan(): CanHook {
  const { data: user } = useUser();
  const { tenant, membership } = useTenant();
  const { billing: billing } = useBilling();
  const meta = useApiMeta();

  const canWithReason = useCallback(
    (evalFn: Evaluate) => {
      return evalFn({
        user,
        membership,
        tenant,
        billing,
        meta,
      });
    },
    [user, membership, tenant, billing, meta],
  );

  const can = useCallback(
    (evalFn: Evaluate) => {
      return canWithReason(evalFn).allowed;
    },
    [canWithReason],
  );

  return {
    can,
    canWithReason,
  };
}
