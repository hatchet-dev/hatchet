import { useSyncExternalStore } from 'react';

let isOpen = false;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

export function setFeedbackDialogOpen(next: boolean) {
  if (isOpen === next) {
    return;
  }
  isOpen = next;
  emit();
}

export function openFeedbackDialog() {
  setFeedbackDialogOpen(true);
}

export function useFeedbackDialogOpen() {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    () => isOpen,
    () => isOpen,
  );
}
