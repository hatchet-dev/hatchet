import { useBananaMode, useBananaRain } from './use-banana-mode';
import { Switch } from '@/components/v1/ui/switch';
import { SettingRow } from '@/pages/main/v1/tenant-settings/components/settings-row';

export function BananaModeSwitch() {
  const [enabled, setEnabled] = useBananaMode();
  const [rain, setRain] = useBananaRain();

  return (
    <>
      <SettingRow
        label="Banana mode"
        description={
          <>
            Paint the app yellow. This stays in this browser. See{' '}
            <a
              href="https://hatchet.run/bananas"
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-foreground"
            >
              hatchet.run/bananas
            </a>
            .
          </>
        }
      >
        <Switch
          checked={enabled}
          aria-label="Banana mode"
          onCheckedChange={setEnabled}
        />
      </SettingRow>
      {enabled ? (
        <SettingRow
          label="Raining bananas"
          description="Drop bananas across the app."
        >
          <Switch
            checked={rain}
            aria-label="Raining bananas"
            onCheckedChange={setRain}
          />
        </SettingRow>
      ) : null}
    </>
  );
}
