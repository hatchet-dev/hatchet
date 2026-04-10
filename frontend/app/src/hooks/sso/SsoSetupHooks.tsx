"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { PROVIDER_CONFIG } from "@/lib/sso/sso-constants";
import { createFormSchema, editFormSchema, FormValues } from "@/lib/sso/sso-schemas";
import { IdpInfoFromCustomer, ProviderKey, SsoApi, SsoSetupStep } from "@/lib/sso/sso-types";
import { hydrateSsoForm, inferSsoProvider, normalizeSsoApi, toSsoIdpInfo } from "@/lib/sso/sso-utils";

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
    onSave: () => Promise<void>;
    onDelete: () => Promise<void>;
    handleProviderSelect: (p: ProviderKey) => void;
    handleBack: () => void;
    reset: () => void;
};

const SsoSetupContext = createContext<SsoSetupContextValue | null>(null);

export type SsoSetupProviderProps = {
    children: React.ReactNode;
    api?: Partial<SsoApi>;
    redirectUrl: string;
    onSave?: (data: IdpInfoFromCustomer) => void;
    onDelete?: () => void;
};

export function SsoSetupProvider({
    children,
    api,
    redirectUrl,
    onSave: onSaveCallback,
    onDelete: onDeleteCallback,
}: SsoSetupProviderProps) {
    const [provider, setProvider] = useState<ProviderKey>("Generic");
    const [form, setForm] = useState<FormValues>(() => PROVIDER_CONFIG.Generic.defaultForm as FormValues);

    const [step, setStep] = useState<SsoSetupStep>(SsoSetupStep.ProviderSelection);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const [apiError, setApiError] = useState<string | null>(null);
    const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

    const [idpConfiguration, setIdpConfiguration] = useState<IdpInfoFromCustomer | null>(null);
    const [shouldValidate, setShouldValidate] = useState(false);
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [isOpen, setIsOpen] = useState(false);

    const safeApi = useMemo(() => normalizeSsoApi(api), [api]);

    // Load existing configuration
    useEffect(() => {
        let mounted = true;
        (async () => {
            const res = await safeApi.get();
            console.log(res)
            if (!mounted) return;
            if (res.ok && res.data.idpInfoFromCustomer) {
                const existingIdpInfo = res.data.idpInfoFromCustomer;
                setIdpConfiguration(existingIdpInfo);
                const inferredProvider = inferSsoProvider(existingIdpInfo);
                setProvider(inferredProvider);
                setForm(hydrateSsoForm(inferredProvider, existingIdpInfo));
                // Go to step 2 if there's existing config
                setStep(SsoSetupStep.Configuration);
            }
            setLoading(false);
        })();
        return () => {
            mounted = false;
        };
    }, [safeApi]);

    // Form update handler
    function onChange(key: string, value: string | boolean) {
        setForm((f) => ({ ...f, [key]: value }) as FormValues);
        setHasUnsavedChanges(true);
    }

    // Save handler
    async function onSave() {
        setSaving(true);
        setApiError(null);
        setShouldValidate(true);

        // Use edit schema if we have existing config, create schema for new configs
        const schema = idpConfiguration ? editFormSchema : createFormSchema;
        const parsed = schema.safeParse(form);
        if (!parsed.success) {
            setSaving(false);
            return;
        }

        const idpInfo = toSsoIdpInfo(parsed.data);
        const res = await safeApi.upsert({ idpInfoFromCustomer: idpInfo });

        if (res.ok) {
            setIdpConfiguration(idpInfo);
            setHasUnsavedChanges(false);
            onSaveCallback?.(idpInfo);
            setIsOpen(false); // Close dialog on successful save
        } else {
            setApiError(res.error?.message || "Failed to save");
        }
        setSaving(false);
    }

    // Delete handler
    async function onDelete() {
        setDeleting(true);
        setApiError(null);

        const res = await safeApi.remove();
        if (res.ok) {
            setIdpConfiguration(null);
            onDeleteCallback?.();
            resetToClean();
            setIsOpen(false); // Close dialog on successful delete
        } else {
            setApiError(res.error?.message || "Failed to delete");
        }
        setDeleting(false);
    }

    // Validation errors
    const errors: Record<string, string | undefined> = {};
    if (shouldValidate) {
        const schema = idpConfiguration ? editFormSchema : createFormSchema;
        const v = schema.safeParse(form);
        if (!v.success) {
            for (const issue of v.error.issues) {
                errors[issue.path.join(".")] = issue.message;
            }
        }
    }

    const handleProviderSelect = (p: ProviderKey) => {
        // Don't allow provider change if there's an existing configuration
        if (idpConfiguration) return;

        setProvider(p);
        setForm(PROVIDER_CONFIG[p].defaultForm as FormValues);
        setStep(SsoSetupStep.Configuration);
    };

    const handleBack = () => {
        setStep(SsoSetupStep.ProviderSelection);
        setShouldValidate(false);
        setApiError(null);
    };

    const reset = () => {
        setStep(idpConfiguration ? SsoSetupStep.Configuration : SsoSetupStep.ProviderSelection);
        setHasUnsavedChanges(false);
        setShouldValidate(false);
        setApiError(null);
        setShowDeleteConfirm(false);

        if (idpConfiguration) {
            // Restore to existing config
            const inferred = inferSsoProvider(idpConfiguration);
            setProvider(inferred);
            setForm(hydrateSsoForm(inferred, idpConfiguration));
        } else {
            // No config exists, reset to clean state
            setProvider("Generic");
            setForm(PROVIDER_CONFIG.Generic.defaultForm as FormValues);
        }
    };

    const resetToClean = () => {
        setStep(SsoSetupStep.ProviderSelection);
        setHasUnsavedChanges(false);
        setShouldValidate(false);
        setApiError(null);
        setShowDeleteConfirm(false);
        setProvider("Generic");
        setForm(PROVIDER_CONFIG.Generic.defaultForm as FormValues);
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

    return <SsoSetupContext.Provider value={value}>{children}</SsoSetupContext.Provider>;
}

function useSsoSetupContext() {
    const context = useContext(SsoSetupContext);
    if (!context) {
        throw new Error("useSsoSetupContext must be used within a SsoSetupProvider");
    }
    return context;
}

// Hook for dialog wrapper metadata
export function useSsoSetupFormMetadata() {
    const { loading, hasUnsavedChanges, reset, isOpen, setIsOpen, idpConfiguration } = useSsoSetupContext();
    return { loading, hasUnsavedChanges, reset, isOpen, setIsOpen, hasExistingConfig: !!idpConfiguration };
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
    const { step, idpConfiguration, saving, handleBack, onSave, setShowDeleteConfirm } = useSsoSetupContext();

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
    const { showDeleteConfirm, setShowDeleteConfirm, onDelete, deleting, provider } = useSsoSetupContext();
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
