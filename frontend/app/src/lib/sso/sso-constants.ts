/* ============================================================================
 * Constants & Configuration
 * ========================================================================= */

export const PROVIDER_CONFIG = {
  Okta: {
    displayName: 'Okta',
    defaultForm: {
      provider: 'Okta',
      clientId: '',
      clientSecret: '',
      ssoDomain: '',
      usesPkce: true,
    },
    docsUrl: 'https://docs.byo.propelauth.com/sso/example-setup-guides/okta',
  },
  MicrosoftEntra: {
    displayName: 'Microsoft Entra',
    defaultForm: {
      provider: 'MicrosoftEntra',
      clientId: '',
      clientSecret: '',
      tenantId: '',
      usesPkce: true,
    },
    docsUrl: 'https://docs.byo.propelauth.com/sso/example-setup-guides/entra',
  },
  Google: {
    displayName: 'Google',
    defaultForm: {
      provider: 'Google',
      clientId: '',
      clientSecret: '',
      authUrl: 'https://accounts.google.com/o/oauth2/v2/auth',
      tokenUrl: 'https://oauth2.googleapis.com/token',
      userinfoUrl: 'https://openidconnect.googleapis.com/v1/userinfo',
      usesPkce: true,
    },
    docsUrl: 'https://docs.byo.propelauth.com/sso/example-setup-guides/google',
  },
  OneLogin: {
    displayName: 'OneLogin',
    defaultForm: {
      provider: 'OneLogin',
      clientId: '',
      clientSecret: '',
      authUrl: '',
      tokenUrl: '',
      userinfoUrl: '',
      usesPkce: true,
    },
    docsUrl:
      'https://docs.byo.propelauth.com/sso/example-setup-guides/onelogin',
  },
  JumpCloud: {
    displayName: 'JumpCloud',
    defaultForm: {
      provider: 'JumpCloud',
      clientId: '',
      clientSecret: '',
      authUrl: 'https://oauth.id.jumpcloud.com/oauth2/auth',
      tokenUrl: 'https://oauth.id.jumpcloud.com/oauth2/token',
      userinfoUrl: 'https://oauth.id.jumpcloud.com/userinfo',
      usesPkce: true,
    },
    docsUrl:
      'https://docs.byo.propelauth.com/sso/example-setup-guides/jumpcloud',
  },
  Generic: {
    displayName: 'Generic',
    defaultForm: {
      provider: 'Generic',
      clientId: '',
      clientSecret: '',
      authUrl: '',
      tokenUrl: '',
      userinfoUrl: '',
      usesPkce: true,
    },
    docsUrl:
      'https://docs.byo.propelauth.com/sso/example-setup-guides/overview',
  },
} as const;
