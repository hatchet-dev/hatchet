import { createContext, useContext, useState, useMemo } from 'react';
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
    const savedExpandedState = localStorage.getItem(EXPANDED_STATE_KEY);
    return savedExpandedState ? JSON.parse(savedExpandedState) : false;
  });

  const openProps = useMemo(() => {
    const sheetParam = searchParams.get(SHEET_PARAM_KEY);
    if (sheetParam) {
      return decodeSheetProps(sheetParam) as OpenSheetProps | undefined;
    }
    return undefined;
  }, [searchParams]);

  const updateUrlParams = (props?: OpenSheetProps) => {
    const params = new URLSearchParams(searchParams.toString());
    
    if (props) {
      params.set(SHEET_PARAM_KEY, encodeSheetProps(props));
    } else {
      params.delete(SHEET_PARAM_KEY);
    }
    
    setSearchParams(params);
  };

  const openSheet = (props: OpenSheetProps) => {
    updateUrlParams(props);
  };

  const closeSheet = () => {
    updateUrlParams();
  };

  const toggleSheet = (props: OpenSheetProps) => {
    if (openProps) {
      closeSheet();
    } else {
      openSheet(props);
    }
  };

  const toggleExpand = () => {
    const newExpandedState = !isExpanded;
    localStorage.setItem(EXPANDED_STATE_KEY, JSON.stringify(newExpandedState));
    setIsExpanded(newExpandedState);
  };

  const sheet: SideSheet = {
    isExpanded,
    openProps,
  };

  return {
    sheet,
    open: openSheet,
    toggle: toggleSheet,
    close: closeSheet,
    toggleExpand,
  };
}
