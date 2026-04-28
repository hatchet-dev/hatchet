import { SsoField } from './SsoField';
import { SsoFormInput } from './SsoFormInput';
import { SsoPkceRow } from './SsoPkceRow';
import { useSsoProviderForm } from '@/hooks/sso/SsoSetupHooks';

export function SsoProviderForm() {
  const { form, errors, onChange, isEditMode } = useSsoProviderForm();
  // Common fields for all providers
  const commonSsoFields = (
    <>
      <SsoField label="Client ID" htmlFor="clientId" required>
        <SsoFormInput
          id="clientId"
          placeholder="client_id"
          value={form.clientId}
          onChange={(v) => onChange('clientId', v)}
          error={errors.clientId}
          readOnly={isEditMode}
          autoComplete="off"
          data-1p-ignore
          data-bwignore
          data-lpignore="true"
          data-protonpass-ignore="true"
          data-form-type="other"
        />
      </SsoField>
      <SsoField
        label="Client Secret"
        htmlFor="clientSecret"
        required={!isEditMode}
      >
        <SsoFormInput
          id="clientSecret"
          type="password"
          placeholder={isEditMode ? 'Leave empty to keep existing' : '••••••••'}
          value={form.clientSecret}
          onChange={(v) => onChange('clientSecret', v)}
          error={errors.clientSecret}
          autoComplete="off"
          data-1p-ignore
          data-bwignore
          data-lpignore="true"
          data-protonpass-ignore="true"
          data-form-type="other"
        />
      </SsoField>
    </>
  );

  if (form.provider === 'Okta') {
    return (
      <>
        <SsoField label="SSO Domain" htmlFor="ssoDomain" required>
          <SsoFormInput
            id="ssoDomain"
            placeholder="example.okta.com"
            value={form.ssoDomain || ''}
            onChange={(v) => onChange('ssoDomain', v)}
            error={errors.ssoDomain}
          />
        </SsoField>
        {commonSsoFields}
        <SsoPkceRow
          checked={form.usesPkce}
          onChange={(v) => onChange('usesPkce', v)}
        />
      </>
    );
  }

  if (form.provider === 'MicrosoftEntra') {
    return (
      <>
        <SsoField label="Tenant ID" htmlFor="tenantId" required>
          <SsoFormInput
            id="tenantId"
            placeholder="00000000-0000-0000-0000-000000000000"
            value={form.tenantId || ''}
            onChange={(v) => onChange('tenantId', v)}
            error={errors.tenantId}
          />
        </SsoField>
        {commonSsoFields}
        <SsoPkceRow
          checked={form.usesPkce}
          onChange={(v) => onChange('usesPkce', v)}
        />
      </>
    );
  }

  // Generic providers (Google, OneLogin, JumpCloud, Generic)
  return (
    <>
      {commonSsoFields}
      <SsoField label="Authorize URL" htmlFor="authUrl" required>
        <SsoFormInput
          id="authUrl"
          placeholder="https://.../authorize"
          value={form.authUrl || ''}
          onChange={(v) => onChange('authUrl', v)}
          error={errors.authUrl}
        />
      </SsoField>
      <SsoField label="Token URL" htmlFor="tokenUrl" required>
        <SsoFormInput
          id="tokenUrl"
          placeholder="https://.../token"
          value={form.tokenUrl || ''}
          onChange={(v) => onChange('tokenUrl', v)}
          error={errors.tokenUrl}
        />
      </SsoField>
      <SsoField label="Userinfo URL" htmlFor="userinfoUrl" required>
        <SsoFormInput
          id="userinfoUrl"
          placeholder="https://.../userinfo"
          value={form.userinfoUrl || ''}
          onChange={(v) => onChange('userinfoUrl', v)}
          error={errors.userinfoUrl}
        />
      </SsoField>
      <SsoPkceRow
        checked={form.usesPkce}
        onChange={(v) => onChange('usesPkce', v)}
      />
    </>
  );
}
