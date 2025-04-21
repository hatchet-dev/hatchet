import { useState } from 'react';
import { Button } from '@/next/components/ui/button';
import { Separator } from '@/next/components/ui/separator';
import { Label } from '@/next/components/ui/label';
import { Switch } from '@/next/components/ui/switch';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog';
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/next/components/ui/alert';
import useTenant from '@/next/hooks/use-tenant';
import {
  TenantVersion,
  UpdateTenantRequest,
} from '@/lib/api/generated/data-contracts';
import { UpdateTenantForm } from './components/update-tenant-form';
import { Lock } from 'lucide-react';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Headline, PageTitle } from '@/next/components/ui/page-header';

export default function SettingsOverviewPage() {
  const { tenant } = useTenant();

  if (!tenant) {
    return (
      <div className="flex-grow h-full w-full flex items-center justify-center">
        <p>Loading tenant information...</p>
      </div>
    );
  }

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage your tenant settings">
          General Tenant Settings
        </PageTitle>
      </Headline>
      <Separator className="my-4" />
      <UpdateTenant />
      <Separator className="my-4" />
      <AnalyticsOptOut />
      <Separator className="my-4" />
      <TenantVersionSwitcher />
    </BasicLayout>
  );
}

function UpdateTenant() {
  const { tenant, update } = useTenant();

  if (!tenant) {
    return null;
  }

  return (
    <div className="w-fit">
      <UpdateTenantForm
        isLoading={update.isPending}
        onSubmit={(data: UpdateTenantRequest) => {
          update.mutate(data);
        }}
        tenant={tenant}
      />
    </div>
  );
}

function AnalyticsOptOut() {
  const { tenant, update } = useTenant();
  const [checkedState, setChecked] = useState(false);
  const [changed, setChanged] = useState(false);

  if (!tenant) {
    return null;
  }

  // Set initial state based on tenant data
  if (checkedState !== !!tenant.analyticsOptOut && !changed) {
    setChecked(!!tenant.analyticsOptOut);
  }

  const save = () => {
    update.mutate({ analyticsOptOut: checkedState });
    setChanged(false);
  };

  return (
    <div className="flex flex-col gap-y-2">
      <h2 className="text-xl font-semibold leading-tight text-foreground">
        Analytics Opt-Out
      </h2>
      <p className="text-sm text-muted-foreground">
        Choose whether to opt out of all analytics tracking.
      </p>
      <div className="flex items-center space-x-2">
        <Switch
          id="analytics-opt-out"
          checked={checkedState}
          onCheckedChange={() => {
            setChecked(!checkedState);
            setChanged(true);
          }}
        />
        <Label htmlFor="analytics-opt-out" className="text-sm">
          Analytics Opt-Out
        </Label>
      </div>
      {changed && (
        <Button
          onClick={save}
          className="w-fit mt-2"
          loading={update.isPending}
        >
          Save
        </Button>
      )}
    </div>
  );
}

function TenantVersionSwitcher() {
  const { tenant, update } = useTenant();
  const [showDowngradeModal, setShowDowngradeModal] = useState(false);

  if (!tenant || tenant.version === TenantVersion.V0) {
    return (
      <div>
        This is a v0 tenant. Please upgrade to v1 or use v0 from the frontend.
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-y-2">
      <h2 className="text-xl font-semibold leading-tight text-foreground">
        Tenant Version
      </h2>
      <p className="text-sm text-muted-foreground">
        You can downgrade your tenant to v0 if needed. Please help us improve v1
        by reporting any bugs in our{' '}
        <a
          href="https://github.com/hatchet-dev/hatchet/issues"
          target="_blank"
          rel="noopener noreferrer"
          className="text-indigo-400 hover:underline"
        >
          Github issues.
        </a>
      </p>
      <Button
        onClick={() => setShowDowngradeModal(true)}
        loading={update.isPending}
        variant="destructive"
        className="w-fit"
      >
        Downgrade to v0
      </Button>

      <Dialog open={showDowngradeModal} onOpenChange={setShowDowngradeModal}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Downgrade to v0</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <Alert variant="warning">
              <Lock className="w-4 h-4 mr-2" />
              <AlertTitle>Warning</AlertTitle>
              <AlertDescription>
                Downgrading to v0 will remove access to v1 features and may
                affect your existing workflows. This action should only be taken
                if absolutely necessary.
              </AlertDescription>
            </Alert>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDowngradeModal(false)}
              loading={update.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                update.mutate({ version: TenantVersion.V0 });
                setShowDowngradeModal(false);
              }}
              loading={update.isPending}
            >
              Confirm Downgrade
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
