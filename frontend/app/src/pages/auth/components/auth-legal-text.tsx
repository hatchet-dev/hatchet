export function AuthLegalText() {
  return (
    <p
      data-cy="auth-legal"
      className="w-full text-left text-sm text-gray-500 dark:text-gray-500"
    >
      By clicking continue, you agree to our{' '}
      <a
        href="https://hatchet.run/policies/terms"
        className="underline underline-offset-4 hover:text-primary"
      >
        Terms of Service
      </a>
      ,{' '}
      <a
        href="https://hatchet.run/policies/cookie"
        className="underline underline-offset-4 hover:text-primary"
      >
        Cookie Policy
      </a>
      , and{' '}
      <a
        href="https://hatchet.run/policies/privacy"
        className="underline underline-offset-4 hover:text-primary"
      >
        Privacy Policy
      </a>
      .
    </p>
  );
}
