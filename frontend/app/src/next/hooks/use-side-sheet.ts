import { createContext, useContext, useState, useMemo, useCallback } from 'react';
import { RunDetailSheetSerializableProps } from '@/next/pages/authenticated/dashboard/runs/detail-sheet/run-detail-sheet';
import { useSearchParams } from 'react-router-dom';
import { SHEET_PARAM_KEY, encodeSheetProps, decodeSheetProps } from '@/next/utils/sheet-url';
import { WorkerDetailsProps } from '../pages/authenticated/dashboard/worker-services/components/worker-details';

const EXPANDED_STATE_KEY = 'side-sheet-expanded-state';

export interface SideSheet {
  isExpanded: boolean;
  openProps?: OpenSheetProps;
}

export type OpenSheetProps = {
  type: 'task-detail';
  props: RunDetailSheetSerializableProps;
} | {
  type: 'worker-detail';
  props: WorkerDetailsProps;
}

interface SideSheetContextValue {
  sheet: SideSheet;
  open: (props: OpenSheetProps) => void;
  toggle: (props: OpenSheetProps) => void;
  close: () => void;
  toggleExpand: () => void;
}

// Create a context for the side sheet state
export const SideSheetContext = createContext<SideSheetContextValue | null>(null);

// Hook to be used by consumers to access side sheet context
export function useSideSheet() {
  const context = useContext(SideSheetContext);
  if (!context) {
    throw new Error('useSideSheet must be used within a SideSheetProvider');
  }
  return {
    open: context.open,
    toggle: context.toggle,
    close: context.close,
    sheet: context.sheet,
    toggleExpand: context.toggleExpand,
  };
}

// Hook to create side sheet state (used by the provider only)
export function useSideSheetState(): SideSheetContextValue {
  const [searchParams, setSearchParams] = useSearchParams();
  const [isExpanded, setIsExpanded] = useState<boolean>(() => {
    try {
      const savedExpandedState = localStorage.getItem(EXPANDED_STATE_KEY);
      return savedExpandedState ? JSON.parse(savedExpandedState) : false;
    } catch {
      return false;
    }
  });

  // Memoize decoded sheet properties to prevent unnecessary re-renders
  const openProps = useMemo(() => {
    const sheetParam = searchParams.get(SHEET_PARAM_KEY);
    if (sheetParam) {
      try {
        return decodeSheetProps(sheetParam) as OpenSheetProps | undefined;
      } catch {
        return undefined;
      }
    }
    return undefined;
  }, [searchParams]);

  // Memoize URL parameter update function
  const updateUrlParams = useCallback((props?: OpenSheetProps) => {
    const params = new URLSearchParams(searchParams.toString());

    if (props) {
      params.set(SHEET_PARAM_KEY, encodeSheetProps(props));
    } else {
      params.delete(SHEET_PARAM_KEY);
    }

    setSearchParams(params);
  }, [searchParams, setSearchParams]);

  // Memoize sheet operations
  const openSheet = useCallback((props: OpenSheetProps) => {
    updateUrlParams(props);
  }, [updateUrlParams]);

  const closeSheet = useCallback(() => {
    updateUrlParams();
  }, [updateUrlParams]);

  const toggleSheet = useCallback((props: OpenSheetProps) => {
    if (openProps) {
      closeSheet();
    } else {
      openSheet(props);
    }
  }, [openProps, closeSheet, openSheet]);

  const toggleExpand = useCallback(() => {
    const newExpandedState = !isExpanded;
    try {
      localStorage.setItem(EXPANDED_STATE_KEY, JSON.stringify(newExpandedState));
    } catch {
      // Ignore storage errors
    }
    setIsExpanded(newExpandedState);
  }, [isExpanded]);

  // Memoize sheet state
  const sheet = useMemo<SideSheet>(() => ({
    isExpanded,
    openProps,
  }), [isExpanded, openProps]);

  // Memoize context value
  const contextValue = useMemo<SideSheetContextValue>(() => ({
    sheet,
    open: openSheet,
    toggle: toggleSheet,
    close: closeSheet,
    toggleExpand,
  }), [sheet, openSheet, toggleSheet, closeSheet, toggleExpand]);

  return contextValue;
}
