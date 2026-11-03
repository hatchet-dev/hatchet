import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { Button } from '@/components/v1/ui/button';
import { getLastSupportConsoleUrl } from '@/lib/support-session';
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline';

export default function SupportEnded() {
  const consoleUrl = getLastSupportConsoleUrl();

  return (
    <SetupScreen>
      <SetupCard
        title="Support session ended."
        footer={
          consoleUrl ? (
            <Button asChild size="sm">
              <a href={consoleUrl}>
                Open support console
                <ArrowTopRightOnSquareIcon className="ml-1.5 h-4 w-4" />
              </a>
            </Button>
          ) : undefined
        }
      >
        <p className="text-sm text-muted-foreground">
          Your access to the tenant has been revoked. You can close this tab.
        </p>
      </SetupCard>
    </SetupScreen>
  );
}
