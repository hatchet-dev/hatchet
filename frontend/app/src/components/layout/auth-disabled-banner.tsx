import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';

export function AuthDisabledBanner() {
  return (
    <div
      role="alert"
      className="flex items-center justify-center gap-2 border-b border-red-300 bg-red-600 px-4 py-1.5 text-center text-sm font-medium text-white dark:border-red-800"
    >
      <ExclamationTriangleIcon className="h-4 w-4 flex-shrink-0" />
      You are using an authentication-disabled instance of Hatchet.
    </div>
  );
}
