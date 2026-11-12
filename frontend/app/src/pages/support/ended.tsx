import { SetupCard, SetupScreen } from '@/components/layout/setup-card';

export default function SupportEnded() {
  return (
    <SetupScreen>
      <SetupCard title="Support session ended.">
        <p className="text-sm text-muted-foreground">
          Your access to the tenant has been revoked. Run{' '}
          <code className="font-mono">hat support</code> to open a tenant again.
        </p>
      </SetupCard>
    </SetupScreen>
  );
}
