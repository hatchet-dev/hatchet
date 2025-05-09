import { Link } from 'react-router-dom';
import { Button } from '@/next/components/ui/button';
import { FaGoogle, FaGithub } from 'react-icons/fa';

export function OrContinueWith() {
  return (
    <div className="relative my-4">
      <div className="absolute inset-0 flex items-center">
        <span className="w-full border-t" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-white px-2 text-gray-700 dark:bg-gray-800 dark:text-gray-300">
          Or continue with
        </span>
      </div>
    </div>
  );
}

export function GoogleLogin() {
  return (
    <a href="/api/v1/users/google/start" className="w-full">
      <Button variant="outline" type="button" className="w-full py-2">
        <FaGoogle className="mr-2 h-4 w-4" />
        Google
      </Button>
    </a>
  );
}

export function GithubLogin() {
  return (
    <a href="/api/v1/users/github/start" className="w-full">
      <Button variant="outline" type="button" className="w-full py-2">
        <FaGithub className="mr-2 h-4 w-4" />
        Github
      </Button>
    </a>
  );
}

export function AuthLayout({
  children,
  title,
  prompt,
}: {
  children: React.ReactNode;
  title: string;
  prompt: string;
}) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center w-full h-full lg:flex-row">
      <div className="container relative flex-col items-center justify-center w-full lg:px-0">
        <div className="mx-auto flex w-full max-w-md lg:p-8">
          <div className="flex w-full flex-col justify-center space-y-6">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
              <p className="text-sm text-gray-700 dark:text-gray-300">
                {prompt}
              </p>
            </div>
            {children}
            <p className="text-left text-sm text-gray-700 dark:text-gray-300 w-full">
              By clicking continue, you agree to our{' '}
              <Link
                to="https://www.iubenda.com/terms-and-conditions/76608149"
                className="underline underline-offset-4 hover:text-primary"
              >
                Terms of Service
              </Link>
              ,{' '}
              <Link
                to="https://www.iubenda.com/privacy-policy/76608149/cookie-policy"
                className="underline underline-offset-4 hover:text-primary"
              >
                Cookie Policy
              </Link>
              , and{' '}
              <Link
                to="https://www.iubenda.com/privacy-policy/76608149"
                className="underline underline-offset-4 hover:text-primary"
              >
                Privacy Policy
              </Link>
              .
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
