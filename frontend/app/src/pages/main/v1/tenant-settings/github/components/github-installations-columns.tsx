import { Avatar, AvatarFallback, AvatarImage } from '@/components/v1/ui/avatar';
import { Button } from '@/components/v1/ui/button';
import { GithubAppInstallation } from '@/lib/api/generated/cloud/data-contracts';
import { CheckCircleIcon, PlusCircleIcon } from '@heroicons/react/24/outline';
import { GearIcon } from '@radix-ui/react-icons';

export function GithubAccountCell({
  installation,
}: {
  installation: GithubAppInstallation;
}) {
  return (
    <div className="flex flex-row items-center gap-4">
      <Avatar className="h-6 w-6">
        <AvatarImage src={installation.account_avatar_url} />
        <AvatarFallback />
      </Avatar>
      <div>{installation.account_name}</div>
    </div>
  );
}

export function GithubLinkCell({
  installation,
  onLinkToTenant,
}: {
  installation: GithubAppInstallation;
  onLinkToTenant: (installationId: string) => void;
}) {
  if (installation.is_linked_to_tenant) {
    return (
      <Button
        variant="ghost"
        disabled
        leftIcon={<CheckCircleIcon className="size-4" />}
      >
        Linked
      </Button>
    );
  }
  return installation.type == 'installation' ? (
    <Button
      variant="outline"
      onClick={() => onLinkToTenant(installation.metadata.id)}
      leftIcon={<PlusCircleIcon className="size-4" />}
    >
      Link to tenant
    </Button>
  ) : (
    <a
      href={
        installation.installation_settings_url +
        `&redirect_to=${encodeURIComponent(window.location.pathname)}`
      }
      target="_blank"
      rel="noreferrer"
    >
      <Button variant="outline">Finish Setup</Button>
    </a>
  );
}

export function GithubSettingsCell({
  installation,
}: {
  installation: GithubAppInstallation;
}) {
  return (
    <a
      href={
        installation.installation_settings_url +
        `&redirect_to=${encodeURIComponent(window.location.pathname)}`
      }
      target="_blank"
      rel="noreferrer"
    >
      <Button variant="ghost" leftIcon={<GearIcon className="size-4" />}>
        Configure
      </Button>
    </a>
  );
}
