import { PermissionSet, RejectReason } from '../shared/permission.base';

export interface ComputeType {
  cpuKind: string;
  cpus: number;
  memoryMb: number;
  gpuKind?: string;
  gpus?: number;
}

// Represents the maximum number of worker pools a tenant can create based on their plan
export const workerPoolLimits = {
  free: 1,
  starter: 2,
  growth: 5,
  enterprise: 5,
};

// Represents the maximum number of replicas per worker pool
export const replicaLimits = {
  free: 2,
  starter: 5,
  growth: 20,
  enterprise: 20,
};

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

    return [true, undefined];
  },

  // Check if a tenant can create a new worker pool based on their current count
  canCreateWorkerPool: (currentWorkerPoolCount: number) => (context) => {
    if (!context.billing) {
      return [true, undefined]; // Default to allowing if no billing info
    }

    const plan = context.billing.plan;
    let maxWorkerPools: number;

    switch (plan) {
      case 'free':
        maxWorkerPools = workerPoolLimits.free;
        break;
      case 'starter':
        maxWorkerPools = workerPoolLimits.starter;
        break;
      case 'growth':
        maxWorkerPools = workerPoolLimits.growth;
        break;
      default:
        // For enterprise or unknown plans
        maxWorkerPools = workerPoolLimits.enterprise;
    }

    if (currentWorkerPoolCount >= maxWorkerPools) {
      return [false, RejectReason.UPGRADE_REQUIRED];
    }

    return [true, undefined];
  },

  // Check if the requested number of replicas is allowed for the tenant's plan
  maxReplicas: (replicaCount: number) => (context) => {
    if (!context.billing) {
      return [true, undefined];
    }

    const plan = context.billing.plan;
    let maxReplicas: number;

    switch (plan) {
      case 'free':
        maxReplicas = replicaLimits.free;
        break;
      case 'starter':
        maxReplicas = replicaLimits.starter;
        break;
      case 'growth':
        maxReplicas = replicaLimits.growth;
        break;
      default:
        // For enterprise or unknown plans
        maxReplicas = replicaLimits.enterprise;
    }

    if (replicaCount > maxReplicas) {
      return [false, RejectReason.UPGRADE_REQUIRED];
    }

    return [true, undefined];
  },

  // Check if GPU is allowed for the tenant's plan
  canUseGpu: (gpuConfig: { gpuKind?: string; gpus?: number }) => (context) => {
    if (!context.billing) {
      return [true, undefined];
    }

    // If no GPU is being requested, allow it
    if (!gpuConfig.gpuKind || !gpuConfig.gpus || gpuConfig.gpus === 0) {
      return [true, undefined];
    }

    const plan = context.billing.plan;

    // Only growth and enterprise plans can use GPUs
    switch (plan) {
      case 'free':
      case 'starter':
        return [false, RejectReason.UPGRADE_REQUIRED];
      case 'growth':
        // For growth plan, we might want to limit GPU types or quantities
        // This can be extended with specific GPU restrictions if needed
        return [true, undefined];
      default:
        // Enterprise or unknown plans
        return [true, undefined];
    }
  },

  selectCompute: (machineType: ComputeType) => (context) => {
    // Default to allowing all machine types if no billing context is available
    if (!context.billing) {
      return [true, undefined];
    }

    const plan = context.billing.plan;
    const { cpuKind, cpus, memoryMb, gpuKind, gpus } = machineType;

    // Check GPU restrictions first
    if (gpuKind || gpus) {
      const [gpuAllowed, gpuRejectReason] = managedCompute.canUseGpu({
        gpuKind,
        gpus,
      })(context);
      if (!gpuAllowed) {
        return [false, gpuRejectReason];
      }
    }

    switch (plan) {
      case 'free':
        // Free plan restrictions
        if (cpuKind !== 'shared') {
          return [false, RejectReason.UPGRADE_REQUIRED];
        }
        if (cpus !== 1) {
          return [false, RejectReason.UPGRADE_REQUIRED];
        }
        if (memoryMb > 1024) {
          return [false, RejectReason.UPGRADE_REQUIRED];
        }
        break;
      case 'starter':
        // Starter plan restrictions
        if (cpus > 4) {
          return [false, RejectReason.UPGRADE_REQUIRED];
        }
        if (memoryMb > 4096) {
          return [false, RejectReason.UPGRADE_REQUIRED];
        }
        break;
      case 'growth':
        // Growth plan has fewer restrictions
        // No specific restrictions, they can use any machine type
        break;
      default:
        // For unknown plans or enterprise, default to allowing all types
        // Enterprise plan would fall here, as it's not in the Plan type
        break;
    }

    return [true, undefined];
  },
};
