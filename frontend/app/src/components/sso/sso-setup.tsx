import { SsoDeleteConfirmationDialog } from './SsoDeleteConfirmationDialog';
import { SsoErrorAlert } from './SsoErrorAlert';
import { SsoIdpPicker } from './SsoIdpPicker';
import { SsoLoadingState } from './SsoLoadingState';
import { SsoProviderForm } from './SsoProviderForm';
import { SsoRedirectUriField } from './SsoRedirectUriField';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/v1/ui/dialog';
import {
  SsoSetupProvider,
  useSsoFormActions,
  useSsoFormBody,
  useSsoFormHeader,
  useSsoSetupFormMetadata,
} from '@/hooks/sso/SsoSetupHooks';
import { PROVIDER_CONFIG } from '@/lib/sso/sso-constants';
import { SsoSetupProps, SsoSetupStep } from '@/lib/sso/sso-types';
import { Loader2, ShieldCheck } from 'lucide-react';

export default function SsoSetup({ redirectUrl, api }: SsoSetupProps) {
  return (
    <SsoSetupProvider redirectUrl={redirectUrl} api={api}>
      <SsoSetupDialogWrapper />
    </SsoSetupProvider>
  );
}

function SsoSetupDialogWrapper() {
  const {
    loading,
    hasUnsavedChanges,
    reset,
    isOpen,
    setIsOpen,
    hasExistingConfig,
  } = useSsoSetupFormMetadata();

  // Prevent accidental closing
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen && hasUnsavedChanges) {
      // Trying to close with unsaved changes
      if (
        window.confirm(
          'You have unsaved changes. Are you sure you want to close?',
        )
      ) {
        setIsOpen(false);
        reset();
      }
    } else {
      setIsOpen(newOpen);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange} modal={true}>
      <DialogTrigger asChild>
        <Button variant="outline" disabled={loading} className="cursor-pointer">
          {loading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <ShieldCheck className="mr-2 h-4 w-4" />
          )}
          {loading
            ? 'Loading...'
            : hasExistingConfig
              ? 'Edit SSO Config'
              : 'Set up SSO'}
        </Button>
      </DialogTrigger>
      <DialogContent
        className="!max-w-[80vw] max-h-[90vh] overflow-y-auto sm:!max-w-[600px]"
        onPointerDownOutside={(e) => {
          // Prevent closing on outside click
          e.preventDefault();
        }}
        onEscapeKeyDown={(e) => {
          // Prevent closing on Escape key
          e.preventDefault();
        }}
      >
        <SsoSetupForm />
      </DialogContent>
    </Dialog>
  );
}

function SsoSetupForm() {
  const { loading } = useSsoSetupFormMetadata();

  if (loading) {
    return <SsoLoadingState />;
  }

  return (
    <div className="space-y-6">
      <SsoFormHeader />
      <SsoFormBody />
      <SsoFormActions />
      <SsoDeleteConfirmationDialog />
    </div>
  );
}

function SsoFormHeader() {
  const { step, provider } = useSsoFormHeader();

  return (
    <DialogHeader>
      <DialogTitle>
        Set up Enterprise SSO
        {step === SsoSetupStep.Configuration &&
          ` - ${PROVIDER_CONFIG[provider].displayName}`}
      </DialogTitle>
      {step === SsoSetupStep.ProviderSelection && (
        <DialogDescription>
          Choose your Identity Provider to get started.
        </DialogDescription>
      )}
      {step === SsoSetupStep.Configuration && (
        <DialogDescription>
          Follow{' '}
          <a
            href={PROVIDER_CONFIG[provider].docsUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            these instructions
          </a>{' '}
          to set up your Identity Provider and create an OAuth/OIDC application.
        </DialogDescription>
      )}
    </DialogHeader>
  );
}

function SsoFormBody() {
  const { step, redirectUrl } = useSsoFormBody();

  return (
    <div className="space-y-6">
      {step === SsoSetupStep.ProviderSelection ? (
        <div className="grid gap-4">
          <SsoIdpPicker />
        </div>
      ) : (
        <div className="grid gap-4">
          <SsoRedirectUriField redirectUrl={redirectUrl} />
          <SsoProviderForm />
          <SsoErrorAlert />
        </div>
      )}
    </div>
  );
}

function SsoFormActions() {
  const { step, existing, saving, onBack, onSave, onDeleteClick } =
    useSsoFormActions();

  if (step !== SsoSetupStep.Configuration) {
    return null;
  }

  return (
    <div className="flex w-full items-center justify-between mt-6">
      {!existing && (
        <Button onClick={onBack} className="cursor-pointer">
          Back
        </Button>
      )}
      <div className="flex items-center gap-2 ml-auto">
        {existing && (
          <Button
            variant="destructive"
            onClick={onDeleteClick}
            className="cursor-pointer"
          >
            Delete
          </Button>
        )}
        <Button onClick={onSave} disabled={saving} className="cursor-pointer">
          {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {existing ? 'Save changes' : 'Create OIDC client'}
        </Button>
      </div>
    </div>
  );
}
