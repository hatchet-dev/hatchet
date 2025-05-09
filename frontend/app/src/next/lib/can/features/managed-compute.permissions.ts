import { PermissionSet, RejectReason } from '@/next/lib/can';

interface ComputeType {
  cpuKind: string;
  cpus: number;
  memoryMb: number;
  gpuKind?: string;
  gpus?: number;
}

// Represents the maximum number of worker services a tenant can create based on their plan
const workerServiceLimits = {
  free: 1,
  starter: 2,
  growth: 5,
  enterprise: 5,
};

// Represents the maximum number of replicas per worker service
const replicaLimits = {
  free: 2,
  starter: 5,
  growth: 20,
  enterprise: 20,
};

export const managedCompute: PermissionSet = {
  create: () => (context) => {
    const { meta } = context;

    if (!meta?.isCloud) {
      return {
        allowed: false,
        rejectReason: RejectReason.CLOUD_ONLY,
      };
    }

    const requireBillingForManagedCompute =
      meta.cloud?.requireBillingForManagedCompute;

    if (
      requireBillingForManagedCompute &&
      !context.billing?.hasPaymentMethods
    ) {
      return {
        allowed: false,
        rejectReason: RejectReason.BILLING_REQUIRED,
      };
    }

    return {
      allowed: true,
    };
  },

  // Check if a tenant can create a new worker service based on their current count
  canCreateWorkerService: (currentWorkerServiceCount: number) => (context) => {
    if (!context.billing) {
      return {
        allowed: true,
      }; // Default to allowing if no billing info
    }

    const plan = context.billing.plan;
    let maxWorkerServices: number;

    switch (plan) {
      case 'free':
        maxWorkerServices = workerServiceLimits.free;
        break;
      case 'starter':
        maxWorkerServices = workerServiceLimits.starter;
        break;
      case 'growth':
        maxWorkerServices = workerServiceLimits.growth;
        break;
      default:
        // For enterprise or unknown plans
        maxWorkerServices = workerServiceLimits.enterprise;
    }

    if (currentWorkerServiceCount >= maxWorkerServices) {
      return {
        allowed: false,
        rejectReason: RejectReason.UPGRADE_REQUIRED,
      };
    }

    return {
      allowed: true,
    };
  },

  // Check if the requested number of replicas is allowed for the tenant's plan
  maxReplicas: (replicaCount: number) => (context) => {
    if (!context.billing) {
      return {
        allowed: true,
      };
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
      return {
        allowed: false,
        rejectReason: RejectReason.UPGRADE_REQUIRED,
      };
    }

    return {
      allowed: true,
    };
  },

  // Check if GPU is allowed for the tenant's plan
  canUseGpu: (gpuConfig: { gpuKind?: string; gpus?: number }) => (context) => {
    if (!context.billing) {
      return {
        allowed: true,
      };
    }

    // If no GPU is being requested, allow it
    if (!gpuConfig.gpuKind || !gpuConfig.gpus || gpuConfig.gpus === 0) {
      return {
        allowed: true,
      };
    }

    const plan = context.billing.plan;

    // Only growth and enterprise plans can use GPUs
    switch (plan) {
      case 'free':
      case 'starter':
        return {
          allowed: false,
          rejectReason: RejectReason.UPGRADE_REQUIRED,
        };
      case 'growth':
        // For growth plan, we might want to limit GPU types or quantities
        // This can be extended with specific GPU restrictions if needed
        return {
          allowed: true,
        };
      default:
        // Enterprise or unknown plans
        return {
          allowed: true,
        };
    }
  },

  selectCompute: (machineType: ComputeType) => (context) => {
    // Default to allowing all machine types if no billing context is available
    if (!context.billing) {
      return {
        allowed: true,
      };
    }

    const plan = context.billing.plan;
    const { cpuKind, cpus, memoryMb, gpuKind, gpus } = machineType;

    // Check GPU restrictions first
    if (gpuKind || gpus) {
      const { allowed: gpuAllowed, rejectReason: gpuRejectReason } =
        managedCompute.canUseGpu({
          gpuKind,
          gpus,
        })(context);
      if (!gpuAllowed) {
        return {
          allowed: false,
          rejectReason: gpuRejectReason,
        };
      }
    }

    switch (plan) {
      case 'free':
        // Free plan restrictions
        if (cpuKind !== 'shared') {
          return {
            allowed: false,
            rejectReason: RejectReason.UPGRADE_REQUIRED,
          };
        }
        if (cpus !== 1) {
          return {
            allowed: false,
            rejectReason: RejectReason.UPGRADE_REQUIRED,
          };
        }
        if (memoryMb > 1024) {
          return {
            allowed: false,
            rejectReason: RejectReason.UPGRADE_REQUIRED,
          };
        }
        break;
      case 'starter':
        // Starter plan restrictions
        if (cpus > 4) {
          return {
            allowed: false,
            rejectReason: RejectReason.UPGRADE_REQUIRED,
          };
        }
        if (memoryMb > 4096) {
          return {
            allowed: false,
            rejectReason: RejectReason.UPGRADE_REQUIRED,
          };
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

    return {
      allowed: true,
    };
  },
};
