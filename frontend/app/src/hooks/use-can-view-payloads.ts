import { canViewPayloadsForMember } from '@/lib/payload-permissions';
import { useAppContext } from '@/providers/app-context';
import { useUserUniverse } from '@/providers/user-universe';

/**
 * True when the current user can view task/run payloads (input, output, event data) in the
 * active tenant. Defaults to true while membership hasn't loaded yet.
 */
export default function useCanViewPayloads(): boolean {
  const { tenantId } = useAppContext();
  const { tenantMemberships } = useUserUniverse();

  return canViewPayloadsForMember(
    tenantMemberships?.find((m) => m.tenant?.metadata.id === tenantId),
  );
}
