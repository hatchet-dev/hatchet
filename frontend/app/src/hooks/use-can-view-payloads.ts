import { canViewPayloadsForMember } from '@/lib/payload-permissions';
import { useAppContext } from '@/providers/app-context';
import { useUserUniverse } from '@/providers/user-universe';

export default function useCanViewPayloads(): boolean {
  const { tenantId } = useAppContext();
  const { tenantMemberships } = useUserUniverse();

  return canViewPayloadsForMember(
    tenantMemberships?.find((m) => m.tenant?.metadata.id === tenantId),
  );
}
