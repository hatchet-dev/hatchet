'use client';

import { useSsoConfig } from './useSsoConfig';
import { PROVIDER_CONFIG } from '@/lib/sso/sso-constants';
import {
  createFormSchema,
  editFormSchema,
  FormValues,
} from '@/lib/sso/sso-schemas';
import {
  IdpInfoFromCustomer,
  ProviderKey,
  SsoSetupStep,
} from '@/lib/sso/sso-types';
import {
  hydrateSsoForm,
  inferSsoProvider,
  toSsoIdpInfo,
} from '@/lib/sso/sso-utils';
import { createContext, useContext, useState } from 'react';

export type SsoSetupContextValue = {
  provider: ProviderKey;
  form: FormValues;
  step: SsoSetupStep;
  loading: boolean;
  saving: boolean;
  deleting: boolean;
  apiError: string | null;
  hasUnsavedChanges: boolean;
  idpConfiguration: IdpInfoFromCustomer | null;
  redirectUrl: string;
  errors: Record<string, string | undefined>;
  showDeleteConfirm: boolean;
  setShowDeleteConfirm: (show: boolean) => void;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
  onChange: (key: string, value: string | boolean) => void;
  onSave: () => void;
  onDelete: () => void;
  handleProviderSelect: (p: ProviderKey) => void;
  handleBack: () => void;
  reset: () => void;
};

const SsoSetupContext = createContext<SsoSetupContextValue | null>(null);

export type SsoSetupProviderProps = {
  children: React.ReactNode;
  orgId: string;
  redirectUrl: string;
  onSave?: (data: IdpInfoFromCustomer) => void;
  onDelete?: () => void;
};

export function SsoSetupProvider({
  children,
  orgId,
  redirectUrl,
  onSave: onSaveCallback,
  onDelete: onDeleteCallback,
}: SsoSetupProviderProps) {
  const [draftForm, setDraftForm] = useState<FormValues | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [shouldValidate, setShouldValidate] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isOpen, setIsOpen] = useState(false);

  const { idpConfiguration, loading, upsertMutation, deleteMutation } =
    useSsoConfig(orgId);

  const existingForm = idpConfiguration
    ? hydrateSsoForm(inferSsoProvider(idpConfiguration), idpConfiguration)
    : null;
  const form =
    draftForm ??
    existingForm ??
    (PROVIDER_CONFIG.Generic.defaultForm as FormValues);
  const provider = form.provider as ProviderKey;
  const step =
    idpConfiguration || draftForm
      ? SsoSetupStep.Configuration
      : SsoSetupStep.ProviderSelection;

  const validationSchema = idpConfiguration ? editFormSchema : createFormSchema;

  const saving = upsertMutation.isPending;
  const deleting = deleteMutation.isPending;

  function clearTransientState() {
    setHasUnsavedChanges(false);
    setShouldValidate(false);
    setApiError(null);
  }

  function resetDraftState() {
    setDraftForm(null);
    clearTransientState();
  }

  // Form update handler
  function onChange(key: string, value: string | boolean) {
    setDraftForm(
      (currentForm) =>
        ({
          ...(currentForm ?? form),
          [key]: value,
        }) as FormValues,
    );
    setHasUnsavedChanges(true);
  }

  // Save handler
  function onSave() {
    clearTransientState();
    setShouldValidate(true);

    const parsed = validationSchema.safeParse(form);
    if (!parsed.success) {
      return;
    }

    upsertMutation.mutate(toSsoIdpInfo(parsed.data), {
      onSuccess: (idpInfo) => {
        resetDraftState();
        onSaveCallback?.(idpInfo);
        setIsOpen(false);
      },
      onError: (error) => {
        setApiError(error.message || 'Failed to save');
      },
    });
  }

  // Delete handler
  function onDelete() {
    setApiError(null);
    deleteMutation.mutate(undefined, {
      onSuccess: () => {
        resetDraftState();
        setShowDeleteConfirm(false);
        onDeleteCallback?.();
        setIsOpen(false);
      },
      onError: (error) => {
        setApiError(error.message || 'Failed to delete');
      },
    });
  }

  // Validation errors
  const errors: Record<string, string | undefined> = {};
  if (shouldValidate) {
    const v = validationSchema.safeParse(form);
    if (!v.success) {
      for (const issue of v.error.issues) {
        errors[issue.path.join('.')] = issue.message;
      }
    }
  }

  const handleProviderSelect = (p: ProviderKey) => {
    // Don't allow provider change if there's an existing configuration
    if (idpConfiguration) {
      return;
    }

    setDraftForm(PROVIDER_CONFIG[p].defaultForm as FormValues);
  };

  const handleBack = () => {
    resetDraftState();
  };

  const reset = () => {
    resetDraftState();
    setShowDeleteConfirm(false);
  };

  const value: SsoSetupContextValue = {
    provider,
    form,
    step,
    loading,
    saving,
    deleting,
    apiError,
    hasUnsavedChanges,
    idpConfiguration,
    redirectUrl,
    errors,
    showDeleteConfirm,
    setShowDeleteConfirm,
    isOpen,
    setIsOpen,
    onChange,
    onSave,
    onDelete,
    handleProviderSelect,
    handleBack,
    reset,
  };

  return (
    <SsoSetupContext.Provider value={value}>
      {children}
    </SsoSetupContext.Provider>
  );
}

function useSsoSetupContext() {
  const context = useContext(SsoSetupContext);
  if (!context) {
    throw new Error(
      'useSsoSetupContext must be used within a SsoSetupProvider',
    );
  }
  return context;
}

// Hook for dialog wrapper metadata
export function useSsoSetupFormMetadata() {
  const {
    loading,
    hasUnsavedChanges,
    reset,
    isOpen,
    setIsOpen,
    idpConfiguration,
  } = useSsoSetupContext();
  return {
    loading,
    hasUnsavedChanges,
    reset,
    isOpen,
    setIsOpen,
    hasExistingConfig: !!idpConfiguration,
  };
}

// Hook for FormHeader component
export function useSsoFormHeader() {
  const { step, provider } = useSsoSetupContext();
  return { step, provider };
}

// Hook for FormBody component
export function useSsoFormBody() {
  const { step, form, errors, onChange, redirectUrl } = useSsoSetupContext();
  return { step, form, errors, onChange, redirectUrl };
}

// Hook for FormActions component
export function useSsoFormActions() {
  const {
    step,
    idpConfiguration,
    saving,
    handleBack,
    onSave,
    setShowDeleteConfirm,
  } = useSsoSetupContext();

  const onDeleteClick = () => setShowDeleteConfirm(true);

  return {
    step,
    existing: idpConfiguration,
    saving,
    onBack: handleBack,
    onSave,
    onDeleteClick,
  };
}

// Hook for IdpPicker component
export function useSsoIdpPicker() {
  const { handleProviderSelect } = useSsoSetupContext();
  return { onProviderSelect: handleProviderSelect };
}

// Hook for ProviderForm component
export function useSsoProviderForm() {
  const { form, errors, onChange, idpConfiguration } = useSsoSetupContext();
  return {
    form,
    errors,
    onChange,
    isEditMode: !!idpConfiguration,
  };
}

// Hook for DeleteConfirmationDialog component
export function useSsoDeleteConfirmation() {
  const {
    showDeleteConfirm,
    setShowDeleteConfirm,
    onDelete,
    deleting,
    provider,
  } = useSsoSetupContext();
  return {
    open: showDeleteConfirm,
    onOpenChange: setShowDeleteConfirm,
    onConfirm: onDelete,
    deleting,
    providerName: PROVIDER_CONFIG[provider].displayName,
  };
}

// Hook for ErrorAlert component
export function useSsoErrorAlert() {
  const { apiError } = useSsoSetupContext();
  return { message: apiError };
}
