import { Button } from '@/components/v1/ui/button';
import { useSsoIdpPicker } from '@/hooks/sso/SsoSetupHooks';
import { PROVIDER_CONFIG } from '@/lib/sso/sso-constants';
import { ProviderKey } from '@/lib/sso/sso-types';

export function SsoIdpPicker() {
  const { onProviderSelect } = useSsoIdpPicker();
  const providers = Object.keys(PROVIDER_CONFIG) as ProviderKey[];

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
      {providers.map((p) => (
        <Button
          key={p}
          type="button"
          onClick={() => onProviderSelect(p)}
          className="cursor-pointer"
        >
          <span className="font-medium">{PROVIDER_CONFIG[p].displayName}</span>
        </Button>
      ))}
    </div>
  );
}
