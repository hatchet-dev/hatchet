import { useState } from 'react';
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

export const baseDocsUrl = 'https://docs.hatchet.run';

export function useDocs() {
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
