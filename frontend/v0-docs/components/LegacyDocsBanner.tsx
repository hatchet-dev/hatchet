import React from 'react';
import Link from 'next/link';
import { AlertTriangle } from 'lucide-react';

export const LegacyDocsBanner: React.FC = () => {
  return (
    <div className="w-full py-4 px-6 bg-amber-50 dark:bg-amber-900/20 text-amber-800 dark:text-amber-200">
      <div className="flex items-center justify-center">
        <AlertTriangle className="w-6 h-6 mr-3 flex-shrink-0" />
        <p className="text-base font-medium">
          You're viewing legacy docs for V0 SDKs.{' '}
          <Link href="https://docs.hatchet.run" className="underline font-semibold hover:text-amber-600 dark:hover:text-amber-300">
            View the latest V1 Docs →
          </Link>
        </p>
      </div>
    </div>
  );
};
