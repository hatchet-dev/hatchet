import { createContext, useContext, useState } from 'react';
import docMetadata from '@/next/lib/docs';

export const pages = docMetadata;

export type DocRef = {
  title: string;
  href: string;
};

export interface DocsSheet {
  isOpen: boolean;
  url: string;
  title: string;
}

interface DocsContextValue {
  sheet: DocsSheet;
  open: (doc: DocRef) => void;
  toggle: (doc: DocRef) => void;
  close: () => void;
}

const baseDocsUrl = 'https://docs.hatchet.run';

// Create a context for the docs state
export const DocsContext = createContext<DocsContextValue | null>(null);

// Hook to be used by consumers to access docs context
export function useDocs() {
  const context = useContext(DocsContext);
  if (!context) {
    throw new Error('useDocs must be used within a DocsProvider');
  }
  return {
    open: context.open,
    toggle: context.toggle,
    close: context.close,
    sheet: context.sheet,
  };
}

// Hook to create docs state (used by the provider only)
export function useDocsState(): DocsContextValue {
  const [sheet, setSheet] = useState<DocsSheet>({
    isOpen: false,
    url: '',
    title: '',
  });

  const openSheet = (doc: DocRef) => {
    setSheet({
      isOpen: true,
      url: `${baseDocsUrl}${doc.href}`,
      title: doc.title,
    });
  };

  const closeSheet = () => {
    setSheet((prev) => ({
      ...prev,
      isOpen: false,
    }));
  };

  const toggleSheet = (doc: DocRef) => {
    if (sheet.isOpen) {
      closeSheet();
    } else {
      openSheet(doc);
    }
  };

  return {
    sheet,
    open: openSheet,
    toggle: toggleSheet,
    close: closeSheet,
  };
}
