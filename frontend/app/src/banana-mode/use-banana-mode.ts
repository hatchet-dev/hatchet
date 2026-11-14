import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useOptionalAppContext } from '@/providers/app-context';

const INACTIVE_KEY = 'hatchet:banana-mode:inactive';
const INACTIVE_RAIN_KEY = 'hatchet:banana-rain:inactive';

function useTenantFlag(
  activeKey: (tenantId: string) => string,
  inactiveKey: string,
  defaultValue: boolean,
): [boolean, (enabled: boolean) => void] {
  const tenantId = useOptionalAppContext()?.tenantId;
  const key = tenantId ? activeKey(tenantId) : inactiveKey;
  const [stored, setStored] = useLocalStorageState(key, defaultValue);

  if (!tenantId) {
    return [defaultValue, () => {}];
  }

  return [stored, setStored];
}

export function useBananaMode(): [boolean, (enabled: boolean) => void] {
  return useTenantFlag(
    (tenantId) => `hatchet:banana-mode:${tenantId}`,
    INACTIVE_KEY,
    false,
  );
}

export function useBananaRain(): [boolean, (enabled: boolean) => void] {
  return useTenantFlag(
    (tenantId) => `hatchet:banana-rain:${tenantId}`,
    INACTIVE_RAIN_KEY,
    true,
  );
}
