import { AuthPage } from './auth-page';

export function DirectSignInDisabled() {
  return (
    <AuthPage
      title="Direct sign-in is disabled"
      showLocalAuth={false}
      basicSection={null}
      altAction={null}
    />
  );
}
